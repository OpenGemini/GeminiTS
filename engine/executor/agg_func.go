/*
Copyright 2024 Huawei Cloud Computing Technologies Co., Ltd.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package executor

import "github.com/openGemini/openGemini/lib/util"

func TopCmpByTimeReduce[T util.NumberOnly](a, b *PointItem[T]) bool {
	if a.time != b.time {
		return a.time < b.time
	}
	return a.value > b.value
}

func TopCmpByValueReduce[T util.NumberOnly](a, b *PointItem[T]) bool {
	if a.value != b.value {
		return a.value < b.value
	}
	return a.time > b.time
}

func BottomCmpByValueReduce[T util.NumberOnly](a, b *PointItem[T]) bool {
	if a.value != b.value {
		return a.value > b.value
	}
	return a.time > b.time
}

func BottomCmpByTimeReduce[T util.NumberOnly](a, b *PointItem[T]) bool {
	if a.time != b.time {
		return a.time < b.time
	}
	return a.value < b.value
}

func FrontDiffFunc[T util.NumberOnly](prev, curr T) T {
	return prev - curr
}

func BehindDiffFunc[T util.NumberOnly](prev, curr T) T {
	return curr - prev
}

func AbsoluteDiffFunc[T util.NumberOnly](prev, curr T) T {
	res := prev - curr
	if res >= 0 {
		return res
	}
	return -res
}
