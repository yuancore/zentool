package ztime

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/small-ek/antgo/utils/conv"
)

var (
	ErrLayout = errors.New(`parse layout failed`)
	monthAbbr = [...]string{
		"Jan",
		"Feb",
		"Mar",
		"Apr",
		"May",
		"Jun",
		"Jul",
		"Aug",
		"Sept",
		"Oct",
		"Nov",
		"Dec",
	}
	weekDayAbbr = [...]string{
		"Sun",
		"Mon",
		"Sun",
		"Tue",
		"Wed",
		"Thur",
		"Fri",
		"Sat",
	}
	weekDayChinese = [...]string{
		"星期日",
		"星期一",
		"星期二",
		"星期三",
		"星期四",
		"星期五",
		"星期六",
	}
)

type Times struct {
	Time time.Time
}

// New 创建对象
func New(param ...interface{}) *Times {
	if len(param) > 0 {
		switch r := param[0].(type) {
		case time.Time:
			return WithTime(r)
		case *time.Time:
			return WithTime(*r)
		case string:
			return StrToTime(r)
		case []byte:
			return StrToTime(string(r))
		case int:
			return NewFromTimeStamp(int64(r))
		case int64:
			return NewFromTimeStamp(r)
		default:
			return NewFromTimeStamp(conv.Int64(r))
		}
	}
	return &Times{Time: time.Now()}
}

// 所有常用时间格式常量
var layouts = []string{
	"15:04", "15:04:05",
	"2006-01", "2006-01-02", "2006-01-02 15", "2006-01-02 15:04", "2006-01-02 15:04:05",
	"200601", "20060102", "2006010215", "200601021504", "20060102150405",
	"2006/01", "2006/01/02", "2006/01/02 15", "2006/01/02 15:04", "2006/01/02 15:04:05",
	time.RFC3339, time.RFC1123,
}

// StrToTime String转Time
func StrToTime(str string) *Times {
	if str == "" {
		panic("StrToTime: empty string")
	}

	// 高性能策略：先按长度快速匹配
	switch len(str) {
	case 4: // YYYY
		if y, err := strconv.Atoi(str); err == nil {
			return &Times{Time: time.Date(y, 1, 1, 0, 0, 0, 0, time.Local)}
		}
	case 6: // YYYYMM
		y, m, _, err := parseYMD(str, 6)
		if err == nil {
			return &Times{Time: time.Date(y, time.Month(m), 1, 0, 0, 0, 0, time.Local)}
		}
	case 8: // YYYYMMDD
		if y, m, d, err := parseYMD(str, 8); err == nil {
			return &Times{Time: time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.Local)}
		}
	case 10: // 时间戳秒 或 YYYYMMDDhh
		if ts, err := strconv.ParseInt(str, 10, 64); err == nil {
			return &Times{Time: time.Unix(ts, 0)}
		}
	case 13: // 时间戳毫秒
		if ts, err := strconv.ParseInt(str, 10, 64); err == nil {
			return &Times{Time: time.UnixMilli(ts)}
		}
	case 14: // YYYYMMDDhhmmss
		if y, mo, d, h, mi, s, err := parseFullNumeric(str); err == nil {
			return &Times{Time: time.Date(y, time.Month(mo), d, h, mi, s, 0, time.Local)}
		}
	}

	// 循环尝试标准 layouts
	for _, f := range layouts {
		if t, err := time.ParseInLocation(f, str, time.Local); err == nil {
			return &Times{Time: t}
		}
	}

	// 都不匹配，panic
	panic(fmt.Sprintf("StrToTime: unsupported time format '%s'", str))
}

// parseYMD 辅助解析 YYYYMM 或 YYYYMMDD
func parseYMD(s string, length int) (y, m, d int, err error) {
	if length == 6 {
		y, _ = strconv.Atoi(s[0:4])
		m, _ = strconv.Atoi(s[4:6])
		return y, m, 1, nil
	} else if length == 8 {
		y, _ = strconv.Atoi(s[0:4])
		m, _ = strconv.Atoi(s[4:6])
		d, _ = strconv.Atoi(s[6:8])
		return y, m, d, nil
	}
	return 0, 0, 0, fmt.Errorf("invalid length")
}

// parseFullNumeric 解析 YYYYMMDDhhmmss
func parseFullNumeric(s string) (y, m, d, h, mi, sec int, err error) {
	y, _ = strconv.Atoi(s[0:4])
	m, _ = strconv.Atoi(s[4:6])
	d, _ = strconv.Atoi(s[6:8])
	h, _ = strconv.Atoi(s[8:10])
	mi, _ = strconv.Atoi(s[10:12])
	sec, _ = strconv.Atoi(s[12:14])
	return y, m, d, h, mi, sec, nil
}

// NewFromTimeStamp creates and returns a Time object with given timestamp,
// which can be in seconds to nanoseconds.
// Eg: 1600443866 and 1600443866199266000 are both considered as valid timestamp number.
func NewFromTimeStamp(timestamp int64) *Times {
	if timestamp == 0 {
		return &Times{}
	}
	var sec, nano int64
	if timestamp > 1e9 {
		for timestamp < 1e18 {
			timestamp *= 10
		}
		sec = timestamp / 1e9
		nano = timestamp % 1e9
	} else {
		sec = timestamp
	}
	return WithTime(time.Unix(sec, nano))
}

// Now Initialization time
func Now() *Times {
	timeNow := time.Now()
	return WithTime(timeNow)
}

// WithTime Include time
func WithTime(t time.Time) *Times {
	return &Times{Time: t}
}

// Format ...
func (t *Times) Format(layout string, chinese ...bool) string {
	var c bool
	if len(chinese) > 0 {
		c = chinese[0]
	}
	d, err := t.parseLayout(layout, c)
	if err != nil {
		panic(err)
	}
	return d
}

// padInt 写入至少 width 位前导零的整数，避免 fmt.Sprintf 开销
// padInt writes value zero-padded to at least width digits, avoiding fmt.Sprintf overhead
func padInt(b *strings.Builder, value int, width int) {
	s := strconv.Itoa(value)
	for i := len(s); i < width; i++ {
		b.WriteByte('0')
	}
	b.WriteString(s)
}

// parseLayout [yyyy-MM-dd]->{{.year}}-{{.month}}-{{.day}}
func (t *Times) parseLayout(layout string, chinese bool) (string, error) {
	if len(strings.TrimSpace(layout)) == 0 {
		return "", ErrLayout
	}
	ti := t.Time
	year, monthNumber, dayOfMonth := t.Time.Date()
	thisMonthFirstDay := time.Date(year, monthNumber, 1, 0, 0, 0, 0, time.Local)
	thisYearFirstDay := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
	monthCom := monthNumber.String()
	monthAbbr := monthAbbr[monthNumber-1]
	weekOfMonth := (t.Time.Day()-1)/7 + 1
	weekdayOfThisMonthFirstDay := thisMonthFirstDay.Weekday()
	relativeWeekOfMonth := (dayOfMonth+int(weekdayOfThisMonthFirstDay-1))/7 + 1

	dayOfYear := t.Time.YearDay()
	dayOfWeek := t.Time.Weekday()
	weekOfYear := (dayOfYear+int(thisYearFirstDay.Weekday()-1))/7 + 1
	if weekOfYear/53 >= 1 {
		weekOfYear = 1
	}
	hour, minute, second := t.Time.Clock()
	millisecond := int64(t.Time.Nanosecond()) / 1e6 // 0-999 ms within current second
	rfc822z := ti.Format("-0700")
	stz := ti.Format("MST")
	var am bool
	am = t.Time.Hour() < 12

	var times = new(strings.Builder)
	var i = 0

	for i < len(layout) {
		c := layout[i]
		switch c {
		case 'y': // 年[year]
			y, endIndex := end(i, layout, 'y')
			if length := len(y); length > 3 {
				padInt(times, year, 4)
			} else {
				padInt(times, year%100, 2)
			}
			i = endIndex
		case 'M': // 月[month]
			m, endIndex := end(i, layout, 'M')
			if length := len(m); length > 3 {
				times.WriteString(monthCom)
			} else if length == 3 {
				times.WriteString(monthAbbr)
			} else {
				padInt(times, int(monthNumber), 2)
			}
			i = endIndex
		case 'w': // 年中的周数[number]
			w, endIndex := end(i, layout, 'w')
			padInt(times, weekOfYear, len(w))
			i = endIndex
		case 'W': // 月份中的周数[number]
			W, endIndex := end(i, layout, 'W')
			padInt(times, relativeWeekOfMonth, len(W))
			i = endIndex
		case 'D': // 年中的天数[number]
			D, endIndex := end(i, layout, 'D')
			padInt(times, dayOfYear, len(D))
			i = endIndex
		case 'd': // 月份中的天数[number]
			d, endIndex := end(i, layout, 'd')
			padInt(times, dayOfMonth, len(d))
			i = endIndex
		case 'F': // 月份中的星期[number]
			F, endIndex := end(i, layout, 'F')
			numberLength := 1
			if weekOfMonth > 9 {
				numberLength = 2
			}
			padInt(times, weekOfMonth, len(F)-numberLength+1)
			i = endIndex
		case 'E': // 星期中的天数[text]
			E, endIndex := end(i, layout, 'E')
			if chinese {
				times.WriteString(weekDayChinese[dayOfWeek])
			} else {
				if length := len(E); length > 3 {
					times.WriteString(dayOfWeek.String())
				} else if length == 3 {
					times.WriteString(weekDayAbbr[dayOfWeek])
				} else {
					padInt(times, int(dayOfWeek), length)
				}
			}
			i = endIndex
		case 'a': // am/pm[text]
			_, endIndex := end(i, layout, 'a')
			if chinese {
				if am {
					times.WriteString("上午")
				} else {
					times.WriteString("下午")
				}
			} else {
				if am {
					times.WriteString("AM")
				} else {
					times.WriteString("PM")
				}
			}
			i = endIndex
		case 'H': // 一天中的小时数，0-23[number]
			H, endIndex := end(i, layout, 'H')
			padInt(times, hour, len(H))
			i = endIndex
		case 'k': // 一天中的小时数，1-24[number]
			k, endIndex := end(i, layout, 'k')
			if hour == 0 {
				padInt(times, 1, len(k))
			} else {
				padInt(times, hour, len(k))
			}
			i = endIndex
		case 'K': // am/pm小时数，0-11[number]
			K, endIndex := end(i, layout, 'K')
			padInt(times, hour%12, len(K))
			i = endIndex
		case 'h': // am/pm小时数,1-12[number]
			h, endIndex := end(i, layout, 'h')
			if hour == 0 {
				padInt(times, 1, len(h))
			} else {
				padInt(times, hour%12, len(h))
			}
			i = endIndex
		case 'm': // 小时中的分钟数[number]
			m, endIndex := end(i, layout, 'm')
			padInt(times, minute, len(m))
			i = endIndex
		case 's': // 分钟中的秒数[number]
			s, endIndex := end(i, layout, 's')
			padInt(times, second, len(s))
			i = endIndex
		case 'S': // 毫秒数[number]
			S, endIndex := end(i, layout, 'S')
			padInt(times, int(millisecond), len(S))
			i = endIndex
		case 'z': // 时区（General）
			_, endIndex := end(i, layout, 'z')
			times.WriteString(stz)
			i = endIndex
		case 'Z': // 时区（RFC）
			_, endIndex := end(i, layout, 'Z')
			times.WriteString(rfc822z)
			i = endIndex
		default:
			times.WriteByte(c)
			i = i + 1
		}
	}
	return times.String(), nil
}

// end ...
func end(from int, in string, target rune) (string, int) {
	var out = new(strings.Builder)
	for i := from; i < len(in); i++ {
		r := rune(in[i])
		from = i
		if r == target {
			out.WriteRune(r)
			if i == len(in)-1 {
				return out.String(), i + 1
			}
			continue
		}
		return out.String(), i
	}
	return "", from + 1
}
