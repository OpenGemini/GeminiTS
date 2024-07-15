package query

/*
Copyright (c) 2018 InfluxData
This code is originally from: https://github.com/influxdata/influxdb/blob/1.7/query/compile.go

2022.01.23 It has been modified to compatible files in influx/influxql and influx/query.
2023.06.01 Cancel subquery sortField request
Huawei Cloud Computing Technologies Co., Ltd.
*/

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/influxdata/influxdb/models"
	"github.com/openGemini/openGemini/engine/hybridqp"
	"github.com/openGemini/openGemini/engine/op"
	"github.com/openGemini/openGemini/lib/util/lifted/influx/influxql"
)

// CompileOptions are the customization options for the compiler.
type CompileOptions struct {
	Now time.Time
}

// Statement is a compiled query statement.
type Statement interface {
	// Prepare prepares the statement by mapping shards and finishing the creation
	// of the query plan.
	Prepare(shardMapper ShardMapper, opt SelectOptions) (PreparedStatement, error)
}

// compiledStatement represents a select statement that has undergone some initial processing to
// determine if it is valid and to have some initial modifications done on the AST.
type compiledStatement struct {
	// Condition is the condition used for accessing data.
	Condition influxql.Expr

	// TimeRange is the TimeRange for selecting data.
	TimeRange influxql.TimeRange

	// Interval holds the time grouping interval.
	Interval hybridqp.Interval

	// InheritedInterval marks if the interval was inherited by a parent.
	// If this is set, then an interval that was inherited will not cause
	// a query that shouldn't have an interval to fail.
	InheritedInterval bool

	// ExtraIntervals is the number of extra intervals that will be read in addition
	// to the TimeRange. It is a multiple of Interval and only applies to queries that
	// have an Interval. It is used to extend the TimeRange of the mapped shards to
	// include additional non-emitted intervals used by derivative and other functions.
	// It will be set to the highest number of extra intervals that need to be read even
	// if it doesn't apply to all functions. The number will always be positive.
	// This value may be set to a non-zero value even if there is no interval for the
	// compiled query.
	ExtraIntervals int

	// Ascending is true if the time ordering is ascending.
	Ascending bool

	// FunctionCalls holds a reference to the call expression of every function
	// call that has been encountered.
	FunctionCalls []*influxql.Call

	// OnlySelectors is set to true when there are no aggregate functions.
	OnlySelectors bool

	// HasDistinct is set when the distinct() function is encountered.
	HasDistinct bool

	// FillOption contains the fill option for aggregates.
	FillOption influxql.FillOption

	// TopBottomFunction is set to top or bottom when one of those functions are
	// used in the statement.
	TopBottomFunction string

	// PercentileOGSketchFunction is set to percentile ogsketch when one of those functions are
	// used in the statement.
	PercentileOGSketchFunction string

	// HasAuxiliaryFields is true when the function requires auxiliary fields.
	HasAuxiliaryFields bool

	// Fields holds all of the fields that will be used.
	Fields []*compiledField

	// TimeFieldName stores the name of the time field's column.
	// The column names generated by the compiler will not conflict with
	// this name.
	TimeFieldName string

	// Limit is the number of rows per series this query should be limited to.
	Limit int

	// HasTarget is true if this query is being written into a target.
	HasTarget bool

	// Options holds the configured compiler options.
	Options CompileOptions

	stmt *influxql.SelectStatement
}

func newCompiler(opt CompileOptions) *compiledStatement {
	if opt.Now.IsZero() {
		opt.Now = time.Now().UTC()
	}
	return &compiledStatement{
		OnlySelectors: true,
		TimeFieldName: "time",
		Options:       opt,
	}
}

func Compile(stmt *influxql.SelectStatement, opt CompileOptions) (Statement, error) {
	c := newCompiler(opt)
	c.stmt = stmt
	if err := c.preprocess(c.stmt); err != nil {
		return nil, err
	}
	if err := c.compile(c.stmt); err != nil {
		return nil, err
	}
	c.stmt.TimeAlias = c.TimeFieldName
	c.stmt.Condition = c.Condition

	// Convert TOP/BOTTOM into the TOP(max)/BOTTOM(min)
	c.stmt.RewriteTopBottom()

	// Convert DISTINCT into a call.
	c.stmt.RewriteDistinct()

	// Remove "time" from fields list.
	c.stmt.RewriteTimeFields()

	// Rewrite any regex conditions that could make use of the index.
	c.stmt.RewriteRegexConditions()

	// Convert PERCENTILE_OGSKETCH into the PERCENTILE_APPROX
	c.stmt.RewritePercentileOGSketch()

	if inCond, ok := c.stmt.Condition.(*influxql.InCondition); ok {
		st, err := Compile(inCond.Stmt, CompileOptions{})
		if err != nil {
			return nil, err
		}
		inCond.Stmt = st.(*compiledStatement).stmt
	}
	return c, nil
}

var TimeFilterProtection bool

// preprocess retrieves and records the global attributes of the current statement.
func (c *compiledStatement) preprocess(stmt *influxql.SelectStatement) error {
	c.Ascending = stmt.TimeAscending()
	c.Limit = stmt.Limit
	c.HasTarget = stmt.Target != nil

	valuer := influxql.NowValuer{Now: c.Options.Now, Location: stmt.Location}
	cond, t, err := influxql.ConditionExpr(stmt.Condition, &valuer)
	if err != nil {
		return err
	}
	// Verify that the condition is actually ok to use.
	if err := c.validateCondition(cond); err != nil {
		return err
	}
	c.Condition = cond
	c.TimeRange = t

	// Read the dimensions of the query, validate them, and retrieve the interval
	// if it exists.
	if err := c.compileDimensions(stmt); err != nil {
		return err
	}

	// Retrieve the fill option for the statement.
	c.FillOption = stmt.Fill
	if TimeFilterProtection && c.TimeRange.Min.IsZero() && c.TimeRange.Max.IsZero() {
		return fmt.Errorf("disabled the query because without specifying the time filter")
	}

	// Resolve the min and max times now that we know if there is an interval or not.
	if c.TimeRange.Min.IsZero() {
		c.TimeRange.Min = time.Unix(0, influxql.MinTime).UTC()
	}
	if c.TimeRange.Max.IsZero() {
		// If the interval is non-zero, then we have an aggregate query and
		// need to limit the maximum time to now() for backwards compatibility
		// and usability.
		if !c.Interval.IsZero() {
			c.TimeRange.Max = c.Options.Now
		} else {
			c.TimeRange.Max = time.Unix(0, influxql.MaxTime).UTC()
		}
	}
	return nil
}

func (c *compiledStatement) compile(stmt *influxql.SelectStatement) error {
	if err := c.compileFields(stmt); err != nil {
		return err
	}
	if err := c.validateFields(); err != nil {
		return err
	}

	if err := c.validateUnnestSource(); err != nil {
		return err
	}

	// Look through the sources and compile each of the subqueries (if they exist).
	// We do this after compiling the outside because subqueries may require
	// inherited state.
	for _, source := range stmt.Sources {
		switch source := source.(type) {
		case *influxql.SubQuery:
			source.Statement.OmitTime = true
			if err := c.subquery(source.Statement); err != nil {
				return err
			}
			stmt.SubQueryHasDifferentAscending = source.Statement.SubQueryHasDifferentAscending
		case *influxql.Join:
			if lsrc, ok := source.LSrc.(*influxql.SubQuery); ok {
				lsrc.Statement.OmitTime = true
				if err := c.subquery(lsrc.Statement); err != nil {
					return err
				}
				stmt.SubQueryHasDifferentAscending = lsrc.Statement.SubQueryHasDifferentAscending
			}
			if rsrc, ok := source.RSrc.(*influxql.SubQuery); ok {
				rsrc.Statement.OmitTime = true
				if err := c.subquery(rsrc.Statement); err != nil {
					return err
				}
				if !stmt.SubQueryHasDifferentAscending {
					stmt.SubQueryHasDifferentAscending = rsrc.Statement.SubQueryHasDifferentAscending
				}
			}
		case *influxql.BinOp:
			if lsrc, ok := source.LSrc.(*influxql.SubQuery); ok {
				lsrc.Statement.OmitTime = true
				if err := c.subquery(lsrc.Statement); err != nil {
					return err
				}
			}
			if rsrc, ok := source.RSrc.(*influxql.SubQuery); ok {
				rsrc.Statement.OmitTime = true
				if err := c.subquery(rsrc.Statement); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (c *compiledStatement) compileFields(stmt *influxql.SelectStatement) error {
	valuer := MathValuer{}

	c.Fields = make([]*compiledField, 0, len(stmt.Fields))
	for _, f := range stmt.Fields {
		// Remove any time selection (it is automatically selected by default)
		// and set the time column name to the alias of the time field if it exists.
		// Such as SELECT time, max(value) FROM cpu will be SELECT max(value) FROM cpu
		// and SELECT time AS timestamp, max(value) FROM cpu will return "timestamp"
		// as the column name for the time.
		if ref, ok := f.Expr.(*influxql.VarRef); ok && ref.Val == "time" {
			if f.Alias != "" {
				c.TimeFieldName = f.Alias
			}
			continue
		}

		// Append this field to the list of processed fields and compile it.
		f.Expr = influxql.Reduce(f.Expr, &valuer)
		field := &compiledField{
			global:        c,
			Field:         f,
			AllowWildcard: true,
		}
		c.Fields = append(c.Fields, field)
		if err := field.compileExpr(field.Field.Expr); err != nil {
			return err
		}
	}
	return nil
}

type compiledField struct {
	// This holds the global state from the compiled statement.
	global *compiledStatement

	// Field is the top level field that is being compiled.
	Field *influxql.Field

	// AllowWildcard is set to true if a wildcard or regular expression is allowed.
	AllowWildcard bool
}

// compileExpr creates the node that executes the expression and connects that
// node to the WriteEdge as the output.
func (c *compiledField) compileExpr(expr influxql.Expr) error {
	switch expr := expr.(type) {
	case *influxql.VarRef:
		// A bare variable reference will require auxiliary fields.
		c.global.HasAuxiliaryFields = true
		return nil
	case *influxql.Wildcard:
		// Wildcards use auxiliary fields. We assume there will be at least one
		// expansion.
		c.global.HasAuxiliaryFields = true
		if !c.AllowWildcard {
			return errors.New("unable to use wildcard in a binary expression")
		}
		return nil
	case *influxql.RegexLiteral:
		if !c.AllowWildcard {
			return errors.New("unable to use regex in a binary expression")
		}
		c.global.HasAuxiliaryFields = true
		return nil
	case *influxql.Call:
		if op.IsProjectOp(expr) {
			return c.compileProjectOp(expr)
		}

		if mathFunc := GetMathFunction(expr.Name); mathFunc != nil {
			return mathFunc.CompileFunc(expr, c)
		}

		if stringFunc := GetStringFunction(expr.Name); stringFunc != nil {
			return stringFunc.CompileFunc(expr, c)
		}

		if labelFunc := GetLabelFunction(expr.Name); labelFunc != nil {
			return labelFunc.CompileFunc(expr, c)
		}

		if promTimeFunc := GetPromTimeFunction(expr.Name); promTimeFunc != nil {
			return promTimeFunc.CompileFunc(expr, c)
		}

		// Register the function call in the list of function calls.
		c.global.FunctionCalls = append(c.global.FunctionCalls, expr)

		if op.IsAggregateOp(expr) {
			return c.compileAggregateOp(expr)
		}

		if aggFunc := GetAggregateOperator(expr.Name); aggFunc != nil {
			return aggFunc.CompileFunc(expr, c)
		}

		switch expr.Name {
		case "exponential_moving_average", "double_exponential_moving_average", "triple_exponential_moving_average", "relative_strength_index", "triple_exponential_derivative":
			return c.compileExponentialMovingAverage(expr.Name, expr.Args)
		case "kaufmans_efficiency_ratio", "kaufmans_adaptive_moving_average":
			return c.compileKaufmans(expr.Name, expr.Args)
		case "chande_momentum_oscillator":
			return c.compileChandeMomentumOscillator(expr.Args)
		case "holt_winters", "holt_winters_with_fit":
			withFit := expr.Name == "holt_winters_with_fit"
			return c.compileHoltWinters(expr.Args, withFit)
		default:
			return fmt.Errorf("undefined function %s()", expr.Name)
		}
	case *influxql.Distinct:
		call := expr.NewCall()
		c.global.FunctionCalls = append(c.global.FunctionCalls, call)
		return c.compileDistinct(call.Args, false)
	case *influxql.BinaryExpr:
		// Disallow wildcards in binary expressions. RewriteFields, which expands
		// wildcards, is too complicated if we allow wildcards inside of expressions.
		c.AllowWildcard = false

		// Check if either side is a literal so we only compile one side if it is.
		if _, ok := expr.LHS.(influxql.Literal); ok {
			if _, ok := expr.RHS.(influxql.Literal); ok {
				return errors.New("cannot perform a binary expression on two literals")
			}
			return c.compileExpr(expr.RHS)
		} else if _, ok := expr.RHS.(influxql.Literal); ok {
			return c.compileExpr(expr.LHS)
		} else {
			// Validate both sides of the expression.
			if err := c.compileExpr(expr.LHS); err != nil {
				return err
			}
			if err := c.compileExpr(expr.RHS); err != nil {
				return err
			}
			return nil
		}
	case *influxql.ParenExpr:
		return c.compileExpr(expr.Expr)
	case influxql.Literal:
		return errors.New("field must contain at least one variable")
	}
	return errors.New("unimplemented")
}

// compileNestedExpr ensures that the expression is compiled as if it were
// a nested expression.
func (c *compiledField) compileNestedExpr(expr influxql.Expr) error {
	// Intercept the distinct call so we can pass nested as true.
	switch expr := expr.(type) {
	case *influxql.Call:
		if expr.Name == "distinct" {
			return c.compileDistinct(expr.Args, true)
		}
	case *influxql.Distinct:
		call := expr.NewCall()
		return c.compileDistinct(call.Args, true)
	}
	return c.compileExpr(expr)
}

func (c *compiledField) compileSymbol(name string, field influxql.Expr) error {
	// Must be a variable reference, wildcard, or regexp.
	switch field.(type) {
	case *influxql.VarRef:
		return nil
	case *influxql.Wildcard:
		if !c.AllowWildcard {
			return fmt.Errorf("unsupported expression with wildcard: %s()", name)
		}
		c.global.OnlySelectors = false
		return nil
	case *influxql.RegexLiteral:
		if !c.AllowWildcard {
			return fmt.Errorf("unsupported expression with regex field: %s()", name)
		}
		c.global.OnlySelectors = false
		return nil
	case *influxql.Call:
		if c.global.stmt.Range > 0 {
			return nil
		}
		return fmt.Errorf("expected field argument in %s()", name)
	default:
		return fmt.Errorf("expected field argument in %s()", name)
	}
}

func (c *compiledField) compileExponentialMovingAverage(name string, args []influxql.Expr) error {
	if got := len(args); got < 2 || got > 4 {
		return fmt.Errorf("invalid number of arguments for %s, expected at least 2 but no more than 4, got %d", name, got)
	}

	arg1, ok := args[1].(*influxql.IntegerLiteral)
	if !ok {
		return fmt.Errorf("%s period must be an integer", name)
	} else if arg1.Val < 1 {
		return fmt.Errorf("%s period must be greater than or equal to 1", name)
	}

	if len(args) >= 3 {
		switch arg2 := args[2].(type) {
		case *influxql.IntegerLiteral:
			if name == "triple_exponential_derivative" && arg2.Val < 1 && arg2.Val != -1 {
				return fmt.Errorf("%s hold period must be greater than or equal to 1", name)
			}
			if arg2.Val < 0 && arg2.Val != -1 {
				return fmt.Errorf("%s hold period must be greater than or equal to 0", name)
			}
		default:
			return fmt.Errorf("%s hold period must be an integer", name)
		}
	}

	if len(args) >= 4 {
		switch arg3 := args[3].(type) {
		case *influxql.StringLiteral:
			switch arg3.Val {
			case "exponential", "simple":
			default:
				return fmt.Errorf("%s warmup type must be one of: 'exponential' 'simple'", name)
			}
		default:
			return fmt.Errorf("%s warmup type must be a string", name)
		}
	}

	c.global.OnlySelectors = false
	if c.global.ExtraIntervals < int(arg1.Val) {
		c.global.ExtraIntervals = int(arg1.Val)
	}

	switch arg0 := args[0].(type) {
	case *influxql.Call:
		if c.global.Interval.IsZero() {
			return fmt.Errorf("%s aggregate requires a GROUP BY interval", name)
		}
		return c.compileExpr(arg0)
	default:
		if !c.global.Interval.IsZero() && !c.global.InheritedInterval {
			return fmt.Errorf("aggregate function required inside the call to %s", name)
		}
		return c.compileSymbol(name, arg0)
	}
}

func (c *compiledField) compileKaufmans(name string, args []influxql.Expr) error {
	if got := len(args); got < 2 || got > 3 {
		return fmt.Errorf("invalid number of arguments for %s, expected at least 2 but no more than 3, got %d", name, got)
	}

	arg1, ok := args[1].(*influxql.IntegerLiteral)
	if !ok {
		return fmt.Errorf("%s period must be an integer", name)
	} else if arg1.Val < 1 {
		return fmt.Errorf("%s period must be greater than or equal to 1", name)
	}

	if len(args) >= 3 {
		switch arg2 := args[2].(type) {
		case *influxql.IntegerLiteral:
			if arg2.Val < 0 && arg2.Val != -1 {
				return fmt.Errorf("%s hold period must be greater than or equal to 0", name)
			}
		default:
			return fmt.Errorf("%s hold period must be an integer", name)
		}
	}

	c.global.OnlySelectors = false
	if c.global.ExtraIntervals < int(arg1.Val) {
		c.global.ExtraIntervals = int(arg1.Val)
	}

	switch arg0 := args[0].(type) {
	case *influxql.Call:
		if c.global.Interval.IsZero() {
			return fmt.Errorf("%s aggregate requires a GROUP BY interval", name)
		}
		return c.compileExpr(arg0)
	default:
		if !c.global.Interval.IsZero() && !c.global.InheritedInterval {
			return fmt.Errorf("aggregate function required inside the call to %s", name)
		}
		return c.compileSymbol(name, arg0)
	}
}

func (c *compiledField) compileChandeMomentumOscillator(args []influxql.Expr) error {
	if got := len(args); got < 2 || got > 4 {
		return fmt.Errorf("invalid number of arguments for chande_momentum_oscillator, expected at least 2 but no more than 4, got %d", got)
	}

	arg1, ok := args[1].(*influxql.IntegerLiteral)
	if !ok {
		return fmt.Errorf("chande_momentum_oscillator period must be an integer")
	} else if arg1.Val < 1 {
		return fmt.Errorf("chande_momentum_oscillator period must be greater than or equal to 1")
	}

	if len(args) >= 3 {
		switch arg2 := args[2].(type) {
		case *influxql.IntegerLiteral:
			if arg2.Val < 0 && arg2.Val != -1 {
				return fmt.Errorf("chande_momentum_oscillator hold period must be greater than or equal to 0")
			}
		default:
			return fmt.Errorf("chande_momentum_oscillator hold period must be an integer")
		}
	}

	c.global.OnlySelectors = false
	if c.global.ExtraIntervals < int(arg1.Val) {
		c.global.ExtraIntervals = int(arg1.Val)
	}

	if len(args) >= 4 {
		switch arg3 := args[3].(type) {
		case *influxql.StringLiteral:
			switch arg3.Val {
			case "none", "exponential", "simple":
			default:
				return fmt.Errorf("chande_momentum_oscillator warmup type must be one of: 'none' 'exponential' 'simple'")
			}
		default:
			return fmt.Errorf("chande_momentum_oscillator warmup type must be a string")
		}
	}

	switch arg0 := args[0].(type) {
	case *influxql.Call:
		if c.global.Interval.IsZero() {
			return fmt.Errorf("chande_momentum_oscillator aggregate requires a GROUP BY interval")
		}
		return c.compileExpr(arg0)
	default:
		if !c.global.Interval.IsZero() && !c.global.InheritedInterval {
			return fmt.Errorf("aggregate function required inside the call to chande_momentum_oscillator")
		}
		return c.compileSymbol("chande_momentum_oscillator", arg0)
	}
}

func (c *compiledField) compileHoltWinters(args []influxql.Expr, withFit bool) error {
	name := "holt_winters"
	if withFit {
		name = "holt_winters_with_fit"
	}

	if exp, got := 3, len(args); got != exp {
		return fmt.Errorf("invalid number of arguments for %s, expected %d, got %d", name, exp, got)
	}

	n, ok := args[1].(*influxql.IntegerLiteral)
	if !ok {
		return fmt.Errorf("expected integer argument as second arg in %s", name)
	} else if n.Val <= 0 {
		return fmt.Errorf("second arg to %s must be greater than 0, got %d", name, n.Val)
	}

	s, ok := args[2].(*influxql.IntegerLiteral)
	if !ok {
		return fmt.Errorf("expected integer argument as third arg in %s", name)
	} else if s.Val < 0 {
		return fmt.Errorf("third arg to %s cannot be negative, got %d", name, s.Val)
	}
	c.global.OnlySelectors = false

	call, ok := args[0].(*influxql.Call)
	if !ok {
		return fmt.Errorf("must use aggregate function with %s", name)
	} else if c.global.Interval.IsZero() {
		return fmt.Errorf("%s aggregate requires a GROUP BY interval", name)
	}
	return c.compileNestedExpr(call)
}

func (c *compiledField) compileDistinct(args []influxql.Expr, nested bool) error {
	if len(args) == 0 {
		return errors.New("distinct function requires at least one argument")
	} else if len(args) != 1 {
		return errors.New("distinct function can only have one argument")
	}

	if _, ok := args[0].(*influxql.VarRef); !ok {
		return errors.New("expected field argument in distinct()")
	}
	if !nested {
		c.global.HasDistinct = true
	}
	c.global.OnlySelectors = false
	return nil
}

func (c *compiledField) compileProjectOp(expr *influxql.Call) error {
	if err := op.CompileOp(expr); err != nil {
		return err
	}
	// Compile all the argument expressions that are not just literals.
	for _, arg := range expr.Args {
		if _, ok := arg.(influxql.Literal); ok {
			continue
		}
		if err := c.compileExpr(arg); err != nil {
			return err
		}
	}
	return nil
}

func (c *compiledField) normalizedExpr(expr influxql.Expr) influxql.Expr {
	switch e := expr.(type) {
	case *influxql.Distinct:
		return e.NewCall()
	default:
		return expr
	}
}

func (c *compiledField) validateSelector(expr *influxql.Call) {
	switch expr.Name {
	case "max", "min", "first", "last":
		// top/bottom are not included here since they are not typical functions.
	case "count", "sum", "mean", "median", "mode", "stddev", "spread", "rate", "irate", "absent":
		// These functions are not considered selectors.
		c.global.OnlySelectors = false
	}
}

func (c *compiledField) compileAggregateOp(expr *influxql.Call) error {
	c.validateSelector(expr)

	if err := op.CompileOp(expr); err != nil {
		return err
	}

	if arg, ok := expr.Args[0].(*influxql.Call); ok {
		expr := c.normalizedExpr(arg)
		return c.compileCall(expr.(*influxql.Call))
	} else {
		return c.compileSymbol(expr.Name, expr.Args[0])
	}
}

func (c *compiledField) compileCall(expr *influxql.Call) error {
	if op.IsProjectOp(expr) {
		return c.compileProjectOp(expr)
	}

	if mathFunc := GetMathFunction(expr.Name); mathFunc != nil {
		return mathFunc.CompileFunc(expr, c)
	}

	if stringFunc := GetStringFunction(expr.Name); stringFunc != nil {
		return stringFunc.CompileFunc(expr, c)
	}

	if labelFunc := GetLabelFunction(expr.Name); labelFunc != nil {
		return labelFunc.CompileFunc(expr, c)
	}

	if promTimeFunc := GetPromTimeFunction(expr.Name); promTimeFunc != nil {
		return promTimeFunc.CompileFunc(expr, c)
	}

	if op.IsAggregateOp(expr) {
		return c.compileAggregateOp(expr)
	}

	if aggFunc := GetAggregateOperator(expr.Name); aggFunc != nil {
		return aggFunc.CompileFunc(expr, c)
	}

	switch expr.Name {
	case "exponential_moving_average", "double_exponential_moving_average", "triple_exponential_moving_average", "relative_strength_index", "triple_exponential_derivative":
		return c.compileExponentialMovingAverage(expr.Name, expr.Args)
	case "kaufmans_efficiency_ratio", "kaufmans_adaptive_moving_average":
		return c.compileKaufmans(expr.Name, expr.Args)
	case "chande_momentum_oscillator":
		return c.compileChandeMomentumOscillator(expr.Args)
	case "holt_winters", "holt_winters_with_fit":
		withFit := expr.Name == "holt_winters_with_fit"
		return c.compileHoltWinters(expr.Args, withFit)
	default:
		return fmt.Errorf("undefined function %s()", expr.Name)
	}
}

func (c *compiledStatement) compileDimensions(stmt *influxql.SelectStatement) error {
	for _, d := range stmt.Dimensions {
		// Reduce the expression before attempting anything. Do not evaluate the call.
		expr := influxql.Reduce(d.Expr, nil)

		switch expr := expr.(type) {
		case *influxql.VarRef:
			if strings.ToLower(expr.Val) == "time" {
				return errors.New("time() is a function and expects at least one argument")
			}
		case *influxql.Call:
			// Ensure the call is time() and it has one or two duration arguments.
			// If we already have a duration
			if expr.Name != "time" {
				return errors.New("only time() calls allowed in dimensions")
			} else if got := len(expr.Args); got < 1 || got > 2 {
				return errors.New("time dimension expected 1 or 2 arguments")
			} else if lit, ok := expr.Args[0].(*influxql.DurationLiteral); !ok {
				return errors.New("time dimension must have duration argument")
			} else if c.Interval.Duration != 0 {
				return errors.New("multiple time dimensions not allowed")
			} else {
				c.Interval.Duration = lit.Val
				if len(expr.Args) == 2 {
					switch lit := expr.Args[1].(type) {
					case *influxql.DurationLiteral:
						c.Interval.Offset = lit.Val % c.Interval.Duration
					case *influxql.TimeLiteral:
						c.Interval.Offset = lit.Val.Sub(lit.Val.Truncate(c.Interval.Duration))
					case *influxql.Call:
						if lit.Name != "now" {
							return errors.New("time dimension offset function must be now()")
						} else if len(lit.Args) != 0 {
							return errors.New("time dimension offset now() function requires no arguments")
						}
						now := c.Options.Now
						c.Interval.Offset = now.Sub(now.Truncate(c.Interval.Duration))

						// Use the evaluated offset to replace the argument. Ideally, we would
						// use the interval assigned above, but the query engine hasn't been changed
						// to use the compiler information yet.
						expr.Args[1] = &influxql.DurationLiteral{Val: c.Interval.Offset}
					case *influxql.StringLiteral:
						// If literal looks like a date time then parse it as a time literal.
						if lit.IsTimeLiteral() {
							t, err := lit.ToTimeLiteral(stmt.Location)
							if err != nil {
								return err
							}
							c.Interval.Offset = t.Val.Sub(t.Val.Truncate(c.Interval.Duration))
						} else {
							return errors.New("time dimension offset must be duration or now()")
						}
					default:
						return errors.New("time dimension offset must be duration or now()")
					}
				}
			}
		case *influxql.Wildcard:
		case *influxql.RegexLiteral:
		default:
			return errors.New("only time and tag dimensions allowed")
		}

		// Assign the reduced/changed expression to the dimension.
		d.Expr = expr
	}
	return nil
}

// validateFields validates that the fields are mutually compatible with each other.
// This runs at the end of compilation but before linking.
func (c *compiledStatement) validateFields() error {
	// Validate that at least one field has been selected.
	if len(c.Fields) == 0 {
		return errors.New("at least 1 non-time field must be queried")
	}
	// Ensure there are not multiple calls if top/bottom is present.
	if len(c.FunctionCalls) > 1 && c.TopBottomFunction != "" {
		return fmt.Errorf("selector function %s() cannot be combined with other functions", c.TopBottomFunction)
	} else if len(c.FunctionCalls) == 0 {
		switch c.FillOption {
		case influxql.NoFill:
			return errors.New("fill(none) must be used with a function")
		case influxql.LinearFill:
			return errors.New("fill(linear) must be used with a function")
		}
		if !c.Interval.IsZero() && !c.InheritedInterval {
			return errors.New("GROUP BY requires at least one aggregate function")
		}
	}
	// If a distinct() call is present, ensure there is exactly one function.
	if c.HasDistinct && (len(c.FunctionCalls) != 1 || c.HasAuxiliaryFields) {
		return errors.New("aggregate function distinct() cannot be combined with other functions or fields")
	}
	// Ensure there are not different calls if percentile_ogsketch is present.
	if len(c.FunctionCalls) > 1 && c.PercentileOGSketchFunction != "" {
		for _, call := range c.FunctionCalls {
			if call.Name != "percentile_ogsketch" {
				return fmt.Errorf("selector function %s() cannot be combined with other functions", c.PercentileOGSketchFunction)
			}
		}
	}
	// Validate we are using a selector or raw query if auxiliary fields are required.
	if c.HasAuxiliaryFields {
		if !c.OnlySelectors {
			return fmt.Errorf("mixing aggregate and non-aggregate queries is not supported")
		} else if len(c.FunctionCalls) > 1 {
			return fmt.Errorf("mixing multiple selector functions with tags or fields is not supported")
		}
	}
	return nil
}

// validateCondition verifies that all elements in the condition are appropriate.
// For example, aggregate calls don't work in the condition and should throw an
// error as an invalid expression.
func (c *compiledStatement) validateCondition(expr influxql.Expr) error {
	switch expr := expr.(type) {
	case *influxql.BinaryExpr:
		// Verify each side of the binary expression. We do not need to
		// verify the binary expression itself since that should have been
		// done by influxql.ConditionExpr.
		if err := c.validateCondition(expr.LHS); err != nil {
			return err
		}
		if err := c.validateCondition(expr.RHS); err != nil {
			return err
		}
		return nil
	case *influxql.Call:
		if mathFunc := GetMathFunction(expr.Name); mathFunc == nil {
			return fmt.Errorf("invalid function call in condition: %s", expr)
		}

		// How many arguments are we expecting?
		nargs := 1
		switch expr.Name {
		case "atan2", "pow":
			nargs = 2
		}

		// Did we get the expected number of args?
		if got := len(expr.Args); got != nargs {
			return fmt.Errorf("invalid number of arguments for %s, expected %d, got %d", expr.Name, nargs, got)
		}

		// Are all the args valid?
		for _, arg := range expr.Args {
			if err := c.validateCondition(arg); err != nil {
				return err
			}
		}
		return nil
	default:
		return nil
	}
}

func (c *compiledStatement) validateUnnestFunc(u *influxql.Unnest, call *influxql.Call) error {
	switch call.Name {
	case "match_all":
		if len(call.Args) != 2 {
			return fmt.Errorf("unnest match_all need 2 param")
		}
		if _, ok := call.Args[0].(*influxql.VarRef); !ok {
			return fmt.Errorf("the argument of extract function must be pattern")
		}
		r := regexp.MustCompile(call.Args[0].(*influxql.VarRef).Val)
		if r == nil {
			return fmt.Errorf("the argument of extract function must be pattern")
		}
		n := r.NumSubexp()
		if len(u.Aliases) != n {
			return fmt.Errorf("the number of regular matching fields is not equal to the number of alias fields")
		}
		if len(u.Aliases) != len(u.DstType) {
			return fmt.Errorf("the number of DstType is not equal to the number of alias fields")
		}
		return nil
	default:
		return fmt.Errorf("unnest not support %s", call.Name)
	}
}

func (c *compiledStatement) validateUnnestSource() error {
	if c.stmt == nil {
		return nil
	}
	if len(c.stmt.UnnestSource) > 0 {
		for _, v := range c.stmt.UnnestSource {
			if call, ok := v.Expr.(*influxql.Call); ok {
				if err := c.validateUnnestFunc(v, call); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("parse func is not call")
			}
		}
	}
	return nil
}

// subquery compiles and validates a compiled statement for the subquery using
// this compiledStatement as the parent.
func (c *compiledStatement) subquery(stmt *influxql.SelectStatement) error {
	subquery := newCompiler(c.Options)
	if err := subquery.preprocess(stmt); err != nil {
		return err
	}

	// Substitute now() into the subquery condition. Then use ConditionExpr to
	// validate the expression. Do not store the results. We have no way to store
	// and read those results at the moment.
	valuer := influxql.MultiValuer(
		&influxql.NowValuer{Now: c.Options.Now, Location: stmt.Location},
		&MathValuer{},
	)
	stmt.Condition = influxql.Reduce(stmt.Condition, valuer)

	// If the ordering is different and the sort field was specified for the subquery,
	// throw an error.
	if len(stmt.SortFields) != 0 && subquery.Ascending != c.Ascending {
		// return errors.New("subqueries must be ordered in the same direction as the query itself")
		stmt.SubQueryHasDifferentAscending = true
	}
	subquery.Ascending = c.Ascending

	// Find the intersection between this time range and the parent.
	// If the subquery doesn't have a time range, this causes it to
	// inherit the parent's time range.
	subquery.TimeRange = subquery.TimeRange.Intersect(c.TimeRange)

	// If the fill option is null, set it to none so we don't waste time on
	// null values with a redundant fill iterator.
	if !subquery.Interval.IsZero() && subquery.FillOption == influxql.NullFill {
		subquery.FillOption = influxql.NoFill
	}

	// Inherit the grouping interval if the subquery has none.
	if !c.Interval.IsZero() && subquery.Interval.IsZero() {
		subquery.Interval = c.Interval
		subquery.InheritedInterval = true
	}
	return subquery.compile(stmt)
}

func (c *compiledStatement) RewriteJoinSource() {
	sources := make([]*influxql.Join, 0, 8)
	c.RewriteJoinSourceDFS(&sources, c.stmt.Sources)
	c.stmt.JoinSource = sources
}

func (c *compiledStatement) RewriteJoinSourceDFS(joinSources *[]*influxql.Join, sources influxql.Sources) {
	for i := range sources {
		switch s := sources[i].(type) {
		case *influxql.SubQuery:
			c.RewriteJoinSourceDFS(&s.Statement.JoinSource, s.Statement.Sources)
		case *influxql.Join:
			*joinSources = append(*joinSources, influxql.CloneSource(s).(*influxql.Join))
		}
	}
}

func (c *compiledStatement) RewriteBinOpSource() {
	sources := make([]*influxql.BinOp, 0, 8)
	for _, source := range c.stmt.Sources {
		c.RewriteBinOpSourceDFS(&sources, source)
	}

	c.stmt.BinOpSource = sources
}

func (c *compiledStatement) RewriteBinOpSourceDFS(binOpSources *[]*influxql.BinOp, source influxql.Source) {
	switch s := source.(type) {
	case *influxql.SubQuery:
		for _, subSource := range s.Statement.Sources {
			c.RewriteBinOpSourceDFS(&s.Statement.BinOpSource, subSource)
		}
	case *influxql.BinOp:
		*binOpSources = append(*binOpSources, influxql.CloneSource(s).(*influxql.BinOp))
		c.RewriteBinOpSourceDFS(nil, s.LSrc)
		c.RewriteBinOpSourceDFS(nil, s.RSrc)
	}

}

func (c *compiledStatement) Prepare(shardMapper ShardMapper, sopt SelectOptions) (PreparedStatement, error) {
	// If this is a query with a grouping, there is a bucket limit, and the minimum time has not been specified,
	// we need to limit the possible time range that can be used when mapping shards but not when actually executing
	// the select statement. Determine the shard time range here.
	timeRange := c.TimeRange
	if sopt.MaxBucketsN > 0 && !c.stmt.IsRawQuery && timeRange.MinTimeNano() == influxql.MinTime {
		interval, err := c.stmt.GroupByInterval()
		if err != nil {
			return nil, err
		}

		offset, err := c.stmt.GroupByOffset()
		if err != nil {
			return nil, err
		}

		if interval > 0 {
			// Determine the last bucket using the end time.
			opt := ProcessorOptions{
				Interval: hybridqp.Interval{
					Duration: interval,
					Offset:   offset,
				},
			}
			last, _ := opt.Window(c.TimeRange.MaxTimeNano() - 1)

			// Determine the time difference using the number of buckets.
			// Determine the maximum difference between the buckets based on the end time.
			maxDiff := last - models.MinNanoTime
			if maxDiff/int64(interval) > int64(sopt.MaxBucketsN) {
				timeRange.Min = time.Unix(0, models.MinNanoTime)
			} else {
				timeRange.Min = time.Unix(0, last-int64(interval)*int64(sopt.MaxBucketsN-1))
			}
		}
	}

	// Modify the time range if there are extra intervals and an interval.
	if !c.Interval.IsZero() && c.ExtraIntervals > 0 {
		if c.Ascending {
			newTime := timeRange.Min.Add(time.Duration(-c.ExtraIntervals) * c.Interval.Duration)
			if !newTime.Before(time.Unix(0, influxql.MinTime).UTC()) {
				timeRange.Min = newTime
			} else {
				timeRange.Min = time.Unix(0, influxql.MinTime).UTC()
			}
		} else {
			newTime := timeRange.Max.Add(time.Duration(c.ExtraIntervals) * c.Interval.Duration)
			if !newTime.After(time.Unix(0, influxql.MaxTime).UTC()) {
				timeRange.Max = newTime
			} else {
				timeRange.Max = time.Unix(0, influxql.MaxTime).UTC()
			}
		}
	}

	var isFullSeriesQuery, isSpecificSeriesQuery bool
	if isFullSeriesQuery = hybridqp.IsFullSeriesQuery(c.stmt); isFullSeriesQuery {
		sopt.HintType = hybridqp.FullSeriesQuery
	} else if isSpecificSeriesQuery = hybridqp.IsSpecificSeriesQuery(c.stmt); isSpecificSeriesQuery {
		sopt.HintType = hybridqp.SpecificSeriesQuery
	}

	// Create an iterator creator based on the shards in the cluster.
	shards, err := shardMapper.MapShards(c.stmt.Sources, timeRange, sopt, c.stmt.Condition)
	if err != nil {
		return nil, err
	}

	if len(c.stmt.Sources) >= 2 && c.stmt.Sources.HaveMultiStore() {
		return nil, fmt.Errorf("query across multiple storage engines is not supported")
	}

	// Rewrite wildcards, if any exist.
	// TODO: batchEn := atomic.LoadInt32(&batchMapTypeEn) == 1
	batchEn := true
	mapper := FieldMapper{FieldMapper: shards}
	c.RewriteJoinSource()
	c.RewriteBinOpSource()
	stmt, err := c.stmt.RewriteFields(mapper, batchEn, false)
	if err != nil {
		shards.Close()
		return nil, err
	}

	// Validate if the types are correct now that they have been assigned.
	if err := validateTypes(stmt); err != nil {
		shards.Close()
		return nil, err
	}

	// Determine base options for iterators.
	opt, err := NewProcessorOptionsStmt(stmt, sopt)
	if err != nil {
		shards.Close()
		return nil, err
	}
	if err = hybridqp.VerifyHintStmt(c.stmt, &opt); err != nil {
		shards.Close()
		return nil, err
	}

	if len(shards.GetSeriesKey()) != 0 && (isFullSeriesQuery || isSpecificSeriesQuery) {
		opt.SetHintType(hybridqp.FullSeriesQuery)
		opt.SeriesKey = append(opt.SeriesKey, shards.GetSeriesKey()...)
	}

	opt.StartTime, opt.EndTime = c.TimeRange.MinTimeNano(), c.TimeRange.MaxTimeNano()
	opt.Ascending = c.Ascending

	if sopt.MaxBucketsN > 0 && !stmt.IsRawQuery && c.TimeRange.MinTimeNano() > influxql.MinTime {
		interval, err := stmt.GroupByInterval()
		if err != nil {
			shards.Close()
			return nil, err
		}

		if interval > 0 {
			// Determine the start and end time matched to the interval (may not match the actual times).
			first, _ := opt.Window(opt.StartTime)
			last, _ := opt.Window(opt.EndTime - 1)

			// Determine the number of buckets by finding the time span and dividing by the interval.
			buckets := (last - first + int64(interval)) / int64(interval)
			if int(buckets) > sopt.MaxBucketsN {
				shards.Close()
				return nil, fmt.Errorf("max-select-buckets limit exceeded: (%d/%d)", buckets, sopt.MaxBucketsN)
			}
		}
	}

	columns := stmt.ColumnNames()
	opt.IncQuery, opt.LogQueryCurrId, opt.IterID = sopt.IncQuery, sopt.QueryID, sopt.IterID
	return NewPreparedStatement(stmt, &opt, shards, columns, sopt.MaxPointN, c.Options.Now), nil
}
