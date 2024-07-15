package promql2influxql

import (
	"reflect"
	"testing"
	"time"

	"github.com/openGemini/openGemini/lib/util/lifted/influx/influxql"
	"github.com/prometheus/prometheus/promql/parser"
)

func AggregateExpr(input string) *parser.AggregateExpr {
	expr, err := parser.ParseExpr(input)
	if err != nil {
		panic(err)
	}
	v, ok := expr.(*parser.AggregateExpr)
	if !ok {
		panic("bad input")
	}
	return v
}

func TestTranspiler_transpileAggregateExpr(t1 *testing.T) {
	type fields struct {
		Start          *time.Time
		End            *time.Time
		Step           time.Duration
		Timezone       *time.Location
		Evaluation     *time.Time
		parenExprCount int
	}
	type args struct {
		a *parser.AggregateExpr
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    influxql.Node
		wantErr bool
	}{
		{
			name: "1",
			fields: fields{
				Evaluation: &endTime2,
			},
			args: args{
				a: AggregateExpr(`topk(3, go_gc_duration_seconds_count)`),
			},
			want:    parseInfluxqlByYacc(`SELECT top(value, 3) AS value FROM go_gc_duration_seconds_count WHERE time >= '2023-01-06T06:55:00Z' AND time <= '2023-01-06T07:00:00Z' GROUP BY *`),
			wantErr: false,
		},
		{
			name: "2",
			fields: fields{
				Evaluation: &endTime2,
			},
			args: args{
				a: AggregateExpr(`sum(go_gc_duration_seconds_count) by (container)`),
			},
			want:    parseInfluxqlByYacc(`SELECT sum(value) AS value FROM go_gc_duration_seconds_count WHERE time >= '2023-01-06T06:55:00Z' AND time <= '2023-01-06T07:00:00Z' GROUP BY container`),
			wantErr: false,
		},
		{
			name: "3",
			fields: fields{
				Evaluation: &endTime2,
			},
			args: args{
				a: AggregateExpr(`sum by (endpoint) (topk(1, go_gc_duration_seconds_count) by (container))`),
			},
			want:    parseInfluxqlByYacc(`SELECT sum(value) AS value FROM (SELECT top(value, 1) AS value FROM go_gc_duration_seconds_count WHERE time >= '2023-01-06T06:55:00Z' AND time <= '2023-01-06T07:00:00Z' GROUP BY *) WHERE time >= '2023-01-06T06:55:00Z' AND time <= '2023-01-06T07:00:00Z' GROUP BY endpoint`),
			wantErr: false,
		},
		{
			name: "4",
			fields: fields{
				Evaluation: &endTime2,
			},
			args: args{
				a: AggregateExpr(`sum by (endpoint) (sum(go_gc_duration_seconds_count) by (container))`),
			},
			want:    parseInfluxqlByYacc(`SELECT sum(value) AS value FROM (SELECT sum(value) AS value FROM go_gc_duration_seconds_count WHERE time >= '2023-01-06T06:55:00Z' AND time <= '2023-01-06T07:00:00Z' GROUP BY container) WHERE time >= '2023-01-06T06:55:00Z' AND time <= '2023-01-06T07:00:00Z' GROUP BY endpoint`),
			wantErr: false,
		},
		{
			name: "5",
			fields: fields{
				Start: &startTime2, End: &endTime2, Step: step,
			},
			args: args{
				a: AggregateExpr(`topk(3, go_gc_duration_seconds_count)`),
			},
			want:    parseInfluxqlByYacc(`SELECT top(value, 3) AS value FROM go_gc_duration_seconds_count WHERE time >= '2023-01-06T03:55:00Z' AND time <= '2023-01-06T07:00:00Z' GROUP BY *, time(1m) fill(none)`),
			wantErr: false,
		},
		{
			name: "6",
			fields: fields{
				Start: &startTime2, End: &endTime2, Step: step,
			},
			args: args{
				a: AggregateExpr(`sum(go_gc_duration_seconds_count) by (container)`),
			},
			want:    parseInfluxqlByYacc(`SELECT sum(value) AS value FROM go_gc_duration_seconds_count WHERE time >= '2023-01-06T03:55:00Z' AND time <= '2023-01-06T07:00:00Z' GROUP BY container, time(1m) fill(none)`),
			wantErr: false,
		},
		{
			name: "7",
			fields: fields{
				Start: &startTime2, End: &endTime2, Step: step,
			},
			args: args{
				a: AggregateExpr(`sum by (endpoint) (topk(1, go_gc_duration_seconds_count) by (container))`),
			},
			want:    parseInfluxqlByYacc(`SELECT sum(value) AS value FROM (SELECT top(value, 1) AS value FROM go_gc_duration_seconds_count WHERE time >= '2023-01-06T03:55:00Z' AND time <= '2023-01-06T07:00:00Z' GROUP BY *, time(1m) fill(none)) WHERE time >= '2023-01-06T03:55:00Z' AND time <= '2023-01-06T07:00:00Z' GROUP BY endpoint, time(1m) fill(none)`),
			wantErr: false,
		},
		{
			name: "8",
			fields: fields{
				Start: &startTime2, End: &endTime2, Step: step,
			},
			args: args{
				a: AggregateExpr(`sum by (endpoint) (sum(go_gc_duration_seconds_count) by (container))`),
			},
			want:    parseInfluxqlByYacc(`SELECT sum(value) AS value FROM (SELECT sum(value) AS value FROM go_gc_duration_seconds_count WHERE time >= '2023-01-06T03:55:00Z' AND time <= '2023-01-06T07:00:00Z' GROUP BY container, time(1m) fill(none)) WHERE time >= '2023-01-06T03:55:00Z' AND time <= '2023-01-06T07:00:00Z' GROUP BY endpoint, time(1m) fill(none)`),
			wantErr: false,
		},
		{
			name: "9",
			fields: fields{
				Evaluation: &endTime2,
			},
			args: args{
				a: AggregateExpr(`sum without() (go_gc_duration_seconds_count)`),
			},
			want:    parseInfluxqlByYacc(`SELECT sum(value) AS value FROM go_gc_duration_seconds_count WHERE time >= '2023-01-06T06:55:00Z' AND time <= '2023-01-06T07:00:00Z'`),
			wantErr: false,
		},
		{
			name: "10",
			fields: fields{
				Evaluation: &endTime2,
			},
			args: args{
				a: AggregateExpr(`sum without(endpoint) (go_gc_duration_seconds_count)`),
			},
			want:    parseInfluxqlByYacc(`SELECT sum(value) AS value FROM go_gc_duration_seconds_count WHERE time >= '2023-01-06T06:55:00Z' AND time <= '2023-01-06T07:00:00Z' GROUP BY endpoint`),
			wantErr: false,
		},
		{
			name: "11",
			fields: fields{
				Evaluation: &endTime2,
			},
			args: args{
				a: AggregateExpr(`sum without(endpoint, container) (go_gc_duration_seconds_count)`),
			},
			want:    parseInfluxqlByYacc(`SELECT sum(value) AS value FROM go_gc_duration_seconds_count WHERE time >= '2023-01-06T06:55:00Z' AND time <= '2023-01-06T07:00:00Z' GROUP BY endpoint, container`),
			wantErr: false,
		},
		{
			name: "12",
			fields: fields{
				Evaluation: &endTime2,
			},
			args: args{
				a: AggregateExpr(`sum without(nonexistent) (go_gc_duration_seconds_count)`),
			},
			want:    parseInfluxqlByYacc(`SELECT sum(value) AS value FROM go_gc_duration_seconds_count WHERE time >= '2023-01-06T06:55:00Z' AND time <= '2023-01-06T07:00:00Z' GROUP BY nonexistent`),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Transpiler{
				PromCommand: PromCommand{
					Start:         tt.fields.Start,
					End:           tt.fields.End,
					Step:          tt.fields.Step,
					Timezone:      tt.fields.Timezone,
					Evaluation:    tt.fields.Evaluation,
					LookBackDelta: DefaultLookBackDelta,
				},
				timeRange:      0,
				parenExprCount: tt.fields.parenExprCount,
				timeCondition:  nil,
			}
			t.minT, t.maxT = t.findMinMaxTime(t.newEvalStmt(tt.args.a))
			got, err := t.Transpile(tt.args.a)
			if (err != nil) != tt.wantErr {
				t1.Errorf("transpileAggregateExpr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got.String(), tt.want.String()) {
				t1.Errorf("transpileAggregateExpr() got = %v, want %v", got, tt.want)
			}
		})
	}
}
