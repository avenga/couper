package seetie

import (
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

var validKey = regexp.MustCompile("[a-zA-Z_][a-zA-Z0-9_-]*")

func ExpToMap(ctx *hcl.EvalContext, exp hcl.Expression) (map[string]interface{}, hcl.Diagnostics) {
	val, diags := exp.Value(ctx)
	if setSeverityLevel(diags).HasErrors() {
		return nil, filterErrors(diags)
	}
	result := make(map[string]interface{})
	if val.IsNull() {
		return result, nil
	}

	for k, v := range val.AsValueMap() {
		switch v.Type() {
		case cty.Bool:
			result[k] = v.True()
		case cty.String:
			result[k] = v.AsString()
		case cty.List(cty.String):
			result[k] = toStringSlice(v)
		case cty.Number:
			f, _ := v.AsBigFloat().Float64()
			result[k] = f
		default:
			if isTuple(v) {
				result[k] = toStringSlice(v)
				continue
			}
			// unknown types results in empty string which gets removed later on
			result[k] = ""
		}
	}
	return result, nil
}

func ValuesMapToValue(m url.Values) cty.Value {
	result := make(map[string]interface{})
	for k, v := range m {
		result[k] = v
	}
	return MapToValue(result)
}

func MapToValue(m map[string]interface{}) cty.Value {
	if m == nil {
		return cty.MapValEmpty(cty.String)
	}

	ctyMap := make(map[string]cty.Value)
	for k, v := range m {
		if validKey.MatchString(k) {
			ctyMap[k] = cty.StringVal(ToString(v))
		}
	}

	if len(ctyMap) == 0 {
		return cty.MapValEmpty(cty.String)
	}
	return cty.MapVal(ctyMap)
}

func HeaderToMapValue(headers http.Header) cty.Value {
	ctyMap := make(map[string]cty.Value)
	for k, v := range headers {
		if validKey.MatchString(k) {
			ctyMap[strings.ToLower(k)] = cty.StringVal(v[0]) // TODO: ListVal??
		}
	}
	if len(ctyMap) == 0 {
		return cty.MapValEmpty(cty.String)
	}
	return cty.MapVal(ctyMap)
}

func CookiesToMapValue(cookies []*http.Cookie) cty.Value {
	ctyMap := make(map[string]cty.Value)
	for _, cookie := range cookies {
		ctyMap[cookie.Name] = cty.StringVal(cookie.Value) // TODO: ListVal??
	}

	if len(ctyMap) == 0 {
		return cty.MapValEmpty(cty.String)
	}
	return cty.MapVal(ctyMap)
}

func toStringSlice(src cty.Value) []string {
	var l []string
	for _, s := range src.AsValueSlice() {
		if !s.IsKnown() {
			continue
		}
		l = append(l, ValueToString(s))
	}
	return l
}

var whitespaceRegex = regexp.MustCompile(`^\s*$`)

// ValueToString explicitly drops all other (unknown) types and
// converts non whitespace strings or numbers to its string representation.
func ValueToString(v cty.Value) string {
	switch v.Type() {
	case cty.String:
		str := v.AsString()
		if whitespaceRegex.MatchString(str) {
			return ""
		}
		return str
	case cty.Number:
		n := v.AsBigFloat()
		ni, accuracy := n.Int(nil)
		if accuracy == big.Exact {
			return ni.String()
		}
		return n.String()
	default:
		return ""
	}
}

func SliceToString(sl []interface{}) string {
	var str []string
	for _, s := range sl {
		if result := ToString(s); result != "" {
			str = append(str, result)
		}
	}
	return strings.Join(str, ",")
}

func ToString(s interface{}) string {
	switch s.(type) {
	case []string:
		return strings.Join(s.([]string), ",")
	case []interface{}:
		return SliceToString(s.([]interface{}))
	case string:
		return s.(string)
	case int:
		return strconv.Itoa(s.(int))
	case float64:
		return fmt.Sprintf("%0.f", s)
	case bool:
		if !s.(bool) {
			return "false"
		}
		return "true"
	default:
	}
	return ""
}

// isTuple checks by type name since tuple is not comparable by type.
func isTuple(v cty.Value) bool {
	return v.Type().FriendlyNameForConstraint() == "tuple"
}

func setSeverityLevel(diags hcl.Diagnostics) hcl.Diagnostics {
	for _, d := range diags {
		if d.Summary == "Missing map element" {
			d.Severity = hcl.DiagWarning
		}
	}
	return diags
}

func filterErrors(diags hcl.Diagnostics) hcl.Diagnostics {
	var errs hcl.Diagnostics
	for _, err := range diags.Errs() {
		errs = append(errs, err.(*hcl.Diagnostic))
	}
	return errs
}