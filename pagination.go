package irisx

import (
	"html/template"
	"math"
	urlUtil "net/url"
	"sort"
	"strconv"
)

func Page(ctx *Context, ps int, total int64, showTotal bool, classes string) string {
	if nil == ctx {
		return ""
	}
	piName := "pi"
	url := ctx.Request().RequestURI
	current := ctx.URLParamIntDefault(piName, 0)
	if ps <= 0 {
		return ""
	}
	pageCount := int(math.Ceil(float64(total) / float64(ps)))
	// current := si / ps
	r := `<nav>
  <ul class="` + classes + `">
    <li`
	if !hasPreviousPage(current) {
		r += ` class="disabled"`
	}
	r += `>
      <a href="` + GetPageUrl(url, current-1, pageCount, piName) + `" aria-label="Previous">
        上一页
      </a>
    </li>`
	if pageCount > 0 {
		for _, v := range ComputePages(current, total) {
			if v > pageCount-1 {
				break
			}
			if -1 == v {
				r += `<li`
				if v == current {
					r += ` class="active" `
				}
				r += `><a href="#">...</a>`
			} else {
				r += `<li`
				if v == current {
					r += ` class="active" `
				}
				r += `><a href="` + GetPageUrl(url, v, pageCount, piName) + `">` + strconv.Itoa(v+1) + `</a></li>`
			}

		}
	}
	r += `<li`
	if !hasNextPage(current, ps, total) {
		r += ` class="disabled" `
	}
	r += `><a href="` + GetPageUrl(url, current+1, pageCount, piName) + `" aria-label="Next">
        下一页
      </a>
    </li>
  </ul>`
	if showTotal {
		r += `<span class="pagination pagination-total">共   ` + strconv.Itoa(int(total)) + `   条</span>`
	}
	r += `</nav>`
	return r
}

func ComputePages(current int, totalPage int64) []int {
	total := int(totalPage)
	m0 := []int{0, 1, 2, 3, 4, 5, 6}
	m1 := []int{0, 1, 2, 3, 4, 5, 6, -1, total}
	m2 := []int{0, -1, current - 2, current - 1, current, current + 1, current + 2, -1, total}
	m3 := []int{0, -1, total - 6, total - 5, total - 4, total - 3, total - 2, total - 1, total}
	if total <= 7 {
		return m0
	}
	if current < 5 {
		return m1
	}
	if current > total-5 {
		return m3
	}
	return m2
}

func hasPreviousPage(current int) bool {
	return current > 0
}
func hasNextPage(pi, ps int, total int64) bool {
	if total > 0 {
		return total > int64(pi*ps)
	}
	return false
}

func GetPageUrl(url string, index, pageCount int, siName string) string {
	if index < 0 || index >= pageCount {
		return "#"
	}
	urlStruct, _ := urlUtil.ParseRequestURI(url)
	q := urlStruct.Query()

	q.Set(siName, strconv.Itoa(index))
	urlStruct.RawQuery = q.Encode()
	return urlStruct.String()
}

func selector(options map[int]string, defaultValueObject interface{}, needEmptyOption bool, props string) template.HTML {
	defaultValue := 0
	if defaultValueStr, ok := defaultValueObject.(string); ok {
		defaultValue, _ = strconv.Atoi(defaultValueStr)
	}
	if dvInt, ok := defaultValueObject.(int); ok {
		defaultValue = dvInt
	}
	r := `<select ` + props + `>`
	if needEmptyOption {
		r += `<option></option>`
	}
	var keys sort.IntSlice = make([]int, len(options))
	i := 0
	for k := range options {
		keys[i] = k
		i++
	}
	keys.Sort()
	for _, k := range keys {
		v := options[k]
		r += `<option value="` + strconv.Itoa(k) + `"`
		if k == defaultValue {
			r += ` selected="selected" `
		}
		r += ">"
		r += v + `</option>`
	}
	r += `</select>`
	return template.HTML(r)
}
