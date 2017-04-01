// Code generated by ragel
// @generated

// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
//line parse.rl:1
package interpolate

import "fmt"

//line parse.go:9
const interpolate_start int = 8
const interpolate_first_final int = 8
const interpolate_error int = 0

const interpolate_en_main int = 8

//line parse.rl:8

// Parse parses a string for interpolation.
//
// Variables may be specified anywhere in the string in the format ${foo} or
// ${foo:default} where 'default' will be used if the variable foo was unset.
func Parse(data string) (out String, _ error) {
	var (
		// Ragel variables
		cs  = 0
		p   = 0
		pe  = len(data)
		eof = pe

		idx int
		v   variable
		l   literal
		t   term
	)

//line parse.go:39
	{
		cs = interpolate_start
	}

//line parse.go:44
	{
		if p == pe {
			goto _test_eof
		}
		switch cs {
		case 8:
			goto st_case_8
		case 9:
			goto st_case_9
		case 1:
			goto st_case_1
		case 0:
			goto st_case_0
		case 2:
			goto st_case_2
		case 3:
			goto st_case_3
		case 4:
			goto st_case_4
		case 5:
			goto st_case_5
		case 6:
			goto st_case_6
		case 10:
			goto st_case_10
		case 7:
			goto st_case_7
		}
		goto st_out
	st_case_8:
		switch data[p] {
		case 36:
			goto st1
		case 92:
			goto st7
		}
		goto tr11
	tr11:
//line parse.rl:29
		idx = p
//line parse.rl:47
		l = literal(data[idx : p+1])
//line parse.rl:50
		t = l
		goto st9
	tr14:
//line parse.rl:47
		l = literal(data[idx : p+1])
//line parse.rl:50
		t = l
		goto st9
	tr17:
//line parse.rl:52
		out = append(out, t)
//line parse.rl:29
		idx = p
//line parse.rl:47
		l = literal(data[idx : p+1])
//line parse.rl:50
		t = l
		goto st9
	st9:
		if p++; p == pe {
			goto _test_eof9
		}
	st_case_9:
//line parse.go:111
		switch data[p] {
		case 36:
			goto tr15
		case 92:
			goto tr16
		}
		goto tr14
	tr15:
//line parse.rl:52
		out = append(out, t)
		goto st1
	st1:
		if p++; p == pe {
			goto _test_eof1
		}
	st_case_1:
//line parse.go:128
		if data[p] == 123 {
			goto st2
		}
		goto st0
	st_case_0:
	st0:
		cs = 0
		goto _out
	st2:
		if p++; p == pe {
			goto _test_eof2
		}
	st_case_2:
		if data[p] == 95 {
			goto tr2
		}
		switch {
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto tr2
			}
		case data[p] >= 65:
			goto tr2
		}
		goto st0
	tr2:
//line parse.rl:29
		idx = p
//line parse.rl:34
		v.Name = data[idx : p+1]
		goto st3
	tr4:
//line parse.rl:34
		v.Name = data[idx : p+1]
		goto st3
	st3:
		if p++; p == pe {
			goto _test_eof3
		}
	st_case_3:
//line parse.go:169
		switch data[p] {
		case 46:
			goto st4
		case 58:
			goto st5
		case 95:
			goto tr4
		case 125:
			goto tr6
		}
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto tr4
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto tr4
			}
		default:
			goto tr4
		}
		goto st0
	st4:
		if p++; p == pe {
			goto _test_eof4
		}
	st_case_4:
		if data[p] == 95 {
			goto tr4
		}
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto tr4
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto tr4
			}
		default:
			goto tr4
		}
		goto st0
	st5:
		if p++; p == pe {
			goto _test_eof5
		}
	st_case_5:
		if data[p] == 125 {
			goto tr8
		}
		goto tr7
	tr7:
//line parse.rl:38
		idx = p
//line parse.rl:39
		v.Default = data[idx : p+1]
		v.HasDefault = true

		goto st6
	tr9:
//line parse.rl:39
		v.Default = data[idx : p+1]
		v.HasDefault = true

		goto st6
	st6:
		if p++; p == pe {
			goto _test_eof6
		}
	st_case_6:
//line parse.go:244
		if data[p] == 125 {
			goto tr6
		}
		goto tr9
	tr6:
//line parse.rl:50
		t = v
		goto st10
	tr8:
//line parse.rl:38
		idx = p
//line parse.rl:50
		t = v
		goto st10
	tr10:
//line parse.rl:46
		l = literal(data[p : p+1])
//line parse.rl:50
		t = l
		goto st10
	st10:
		if p++; p == pe {
			goto _test_eof10
		}
	st_case_10:
//line parse.go:270
		switch data[p] {
		case 36:
			goto tr15
		case 92:
			goto tr16
		}
		goto tr17
	tr16:
//line parse.rl:52
		out = append(out, t)
		goto st7
	st7:
		if p++; p == pe {
			goto _test_eof7
		}
	st_case_7:
//line parse.go:287
		goto tr10
	st_out:
	_test_eof9:
		cs = 9
		goto _test_eof
	_test_eof1:
		cs = 1
		goto _test_eof
	_test_eof2:
		cs = 2
		goto _test_eof
	_test_eof3:
		cs = 3
		goto _test_eof
	_test_eof4:
		cs = 4
		goto _test_eof
	_test_eof5:
		cs = 5
		goto _test_eof
	_test_eof6:
		cs = 6
		goto _test_eof
	_test_eof10:
		cs = 10
		goto _test_eof
	_test_eof7:
		cs = 7
		goto _test_eof

	_test_eof:
		{
		}
		if p == eof {
			switch cs {
			case 9, 10:
//line parse.rl:52
				out = append(out, t)
//line parse.go:306
			}
		}

	_out:
		{
		}
	}

//line parse.rl:56

	if cs < 8 {
		return out, fmt.Errorf("cannot parse string %q", data)
	}

	return out, nil
}
