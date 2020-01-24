package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

type fields struct {
	FieldName string
	tags
}

type tags struct {
	IsInt     bool
	Required  bool
	Enum      []string
	Min       string
	Max       string
	ParamName string
	Default   string
}

type api_json struct {
	Url    string
	Auth   bool
	Method string
}

type meta struct {
	Handler string
	In      string
	Out     string
	api_json
	ParamIn []fields
}

type serverStruct struct {
	Srv         map[string][]meta
	Validators  map[string][]fields
	IsInt       bool
	IsParamName bool
}

var (
	servers serverStruct
)

func main() {

	fset := token.NewFileSet()
	node, _ := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)

	out, _ := os.Create(os.Args[2])
	fmt.Fprintln(out, `package `+node.Name.Name)

	fmt.Fprintln(out) // empty line
	fmt.Fprintln(out, `import "net/http"`)
	fmt.Fprintln(out, `import "encoding/json"`)
	fmt.Fprintln(out, `import "errors"`)

	fmt.Fprintln(out) // empty line
	fmt.Fprintln(out,
		`var (
		errorUnknown    = errors.New("unknown method")
		errorBad        = errors.New("bad method")
		errorEmptyLogin = errors.New("login must me not empty")) 

type JsonError struct {`)

	fmt.Fprintln(out, "\tError string `json:\"error\"`")
	fmt.Fprintln(out, "}")

	servers := serverStruct{make(map[string][]meta), make(map[string][]fields), false, false}

	for _, f := range node.Decls {
		switch a := f.(type) {
		case *ast.GenDecl:
			doStruct(out, a, &servers)
		case *ast.FuncDecl:
			doFunc(out, a, &servers)
		default:
			continue
		}
	}

	doServeHTTP(out, &servers)
	doResponses(out, &servers)

}

func doStruct(out *os.File, f *ast.GenDecl, s *serverStruct) {
	var isInt bool
	for _, spec := range f.Specs {
		c_type, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		structType := c_type.Type.(*ast.StructType)
		if !ok {
			continue
		}

		for _, field := range structType.Fields.List {
			if field.Tag != nil {
				tagValue := ""
				if strings.HasPrefix(field.Tag.Value, "`apivalidator:") {
					tagValue = strings.TrimLeft(field.Tag.Value, "`apivalidator:")
				}
				if field.Type.(*ast.Ident).Name == "int" {
					s.IsInt = true
					isInt = true
				} else {
					isInt = false
				}
				if strings.Contains(field.Tag.Value, "paramname") {
					s.IsParamName = true
				}
				f := fields{}
				v := strings.Split(strings.Replace(strings.Trim(tagValue, "/`"), "\"", "", -1), ",")
				for _, value := range v {
					if isInt {
						f.IsInt = true
					} else {
						f.IsInt = false
					}
					if value == "required" {
						f.Required = true
						f.FieldName = field.Names[0].Name
					}
					s := strings.Split(value, "=")
					if len(s) > 1 {
						//check for enum
						enums := strings.Split(s[1], "|")
						if len(enums) > 1 {
							for _, enum := range enums {
								f.FieldName = field.Names[0].Name
								f.Enum = append(f.Enum, enum)
							}

						}

						switch s[0] {
						case "min":
							f.FieldName = field.Names[0].Name
							f.Min = s[1]
						case "max":
							f.FieldName = field.Names[0].Name
							f.Max = s[1]
						case "paramname":
							f.FieldName = field.Names[0].Name
							f.ParamName = s[1]
						case "default":
							f.FieldName = field.Names[0].Name
							f.Default = s[1]

						}
					}
				}
				s.Validators[c_type.Name.Name] = append(s.Validators[c_type.Name.Name], f)
			}
		}
	}

}

func doServeHTTP(out *os.File, s *serverStruct) {

	for key, value := range s.Srv {
		fmt.Fprintln(out, "func (h *"+key+") ServeHTTP(w http.ResponseWriter, r *http.Request) {")
		fmt.Fprintln(out, "switch r.URL.Path {")

		// обработать методы
		for _, j := range value {
			fmt.Fprintln(out, ` case "`+j.Url+`":`)
			fmt.Fprintln(out, ` h.`+strings.ToLower(j.Handler)+`(w, r)`)
		}

		fmt.Fprintln(out, ` default:
		js, _ := json.Marshal(JsonError{errorUnknown.Error()})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write(js)
		return`)
		fmt.Fprintln(out, "}")
		fmt.Fprintln(out, "}")
	}

}

func doResponses(out *os.File, s *serverStruct) {

	for key, value := range s.Srv {
		for _, j := range value {
			fmt.Fprintln(out, ` type Response`+key+j.Handler+` struct {`)
			fmt.Fprintln(out, `*`+j.Out+` `+" `json:\"response\"`")
			fmt.Fprintln(out, `JsonError`)
			fmt.Fprintln(out, `}`)
		}
	}

}

func doFunc(out *os.File, f *ast.FuncDecl, s *serverStruct) {

	if f.Doc == nil {
		return
	}

	currFuncType := ""
	currFuncName := ""

	for _, comment := range f.Doc.List {
		if strings.HasPrefix(comment.Text, "// apigen:api") {

			text_func := "func (h *"

			m := meta{}
			aj := api_json{}

			j := []byte(strings.TrimLeft(comment.Text, "// apigen:api"))
			json.Unmarshal(j, &aj)
			m.api_json = aj
			m.Handler = f.Name.Name

			if f.Type.Params.List != nil {
				for _, p := range f.Type.Params.List {
					switch a := p.Type.(type) {
					case *ast.Ident:
						m.In = a.Name
					}
				}
			}

			if f.Type.Results.List != nil && len(f.Type.Results.List) != 0 {
				switch a := f.Type.Results.List[0].Type.(type) {
				case *ast.StarExpr:
					m.Out = a.X.(*ast.Ident).Name
				}
			}

			type_func := ""
			if f.Recv != nil {
				switch a := f.Recv.List[0].Type.(type) {
				case *ast.StarExpr:
					t := a.X.(*ast.Ident).Name
					currFuncType = t
					s.Srv[t] = append(s.Srv[t], m)
					type_func += t + ") "
				}
			}

			currFuncName = f.Name.Name
			name_func := strings.ToLower(f.Name.Name)

			text_func += type_func + name_func
			param_func := " (w http.ResponseWriter, r *http.Request) "
			text_func += param_func

			body_func := "{\n"
			body_func += "ctx := r.Context() \n"

			text_func += body_func
			fmt.Fprintln(out, text_func)
		}
	}
	body_func := ""
	for key, value := range s.Srv {
		for i, j := range value {
			fmt.Println(i)
			if currFuncType == key && currFuncName == j.Handler {

				params := make(map[string]tags)
				for v1, v2 := range s.Validators {
					fmt.Println(v1)

					var param string
					for k, m := range v2 {
						fmt.Println(k)

						if v1 == j.In {
							//if len(param) > 0 {
							//	param += ","
							//}
							fieldName := m.FieldName
							if len(fieldName) > 0 {
								param = fieldName + ","
								fmt.Println(param)
								params[param] = m.tags
							}
						}
					}
				}

				for s, l := range params {
					body_func += "var " + strings.ToLower(strings.TrimRight(s, ","))
					if l.IsInt {
						body_func += " int\n"
					} else {
						body_func += " string\n"
					}
				}

				if j.Method != "POST" {
					body_func += "switch r.Method { \n"
					body_func += "case \"GET\":\n"
					for s, _ := range params {
						param := strings.ToLower(strings.TrimRight(s, ","))
						body_func += param + " = " + "r.URL.Query().Get(\"" + param + "\")\n"
						body_func += "if " + param + "==\"\" {\n"
						body_func += `js, _ := json.Marshal(JsonError{errorEmptyLogin.Error()})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write(js)
			return
		}`
						body_func += "\n"
						body_func += "case \"POST\":\n"
						for s, _ := range params {
							param := strings.ToLower(strings.TrimRight(s, ","))
							body_func += "r.ParseForm()\n"
							body_func += param + " = " + "r.Form.Get(\"" + param + "\")\n"
							body_func += "if " + param + "==\"\" {\n"
							body_func += `js, _ := json.Marshal(JsonError{errorEmptyLogin.Error()})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write(js)
			return
		}}`
							body_func += "\n"
						}
					}
				} else {
					body_func += `if r.Method != "POST" {
		js, _ := json.Marshal(JsonError{errorBad.Error()})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write(js)
		return
	}
	if r.Header.Get("X-Auth") != "100500" {
		js, _ := json.Marshal(JsonError{"unauthorized"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write(js)
		return
	}
r.ParseForm()`
					body_func += "\n"

					for s, l := range params {
						ff := strings.TrimRight(strings.ToLower(s), ",")
						if l.IsInt {
							body_func += strings.ToLower(ff) + ", err := strconv.Atoi(r.Form.Get(\"" + ff + "\"))\n"
							body_func += "if err != nil {\n"
							body_func += "js, _ := json.Marshal(JsonError{\"" + ff + " must be int\"})\n"
							body_func += `w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}`
							body_func += "\n"

							//body_func += "if " + strings.ToLower(f) + " == nil {\n"

	//						body_func += `js, _ := json.Marshal(JsonError{errorEmptyLogin.Error()})
	//	w.Header().Set("Content-Type", "application/json")
	//	w.WriteHeader(http.StatusBadRequest)
	//	w.Write(js)
	//	return
	//}`
							body_func += "\n"

						} else {
							body_func += strings.ToLower(ff) + " = r.Form.Get(\"" + ff + "\")\n"
							body_func += "\n"
						}

						if l.Required {
							if l.IsInt {
								body_func += "\n if " + strings.ToLower(ff) + " == nil {\n"
								body_func += `js, _ := json.Marshal(JsonError{errorEmptyLogin.Error()})
									w.Header().Set("Content-Type", "application/json")
									w.WriteHeader(http.StatusBadRequest)
									w.Write(js)
									return
								}`
								body_func += "\n"
							} else {
								body_func += "\n if " + strings.ToLower(ff) + " == \"\" {\n"
								body_func += `js, _ := json.Marshal(JsonError{errorEmptyLogin.Error()})
									w.Header().Set("Content-Type", "application/json")
									w.WriteHeader(http.StatusBadRequest)
									w.Write(js)
									return
								}`
								body_func += "\n"
							}
						}

						if len(l.ParamName) > 0 {
							body_func += "paramname_" + ff + " := r.Form.Get(\"" + l.ParamName + "\")\n"
							body_func += "if paramname_" + ff + " == \"\" {\n"
							body_func += ff + " = strings.ToLower(" + ff + ")\n} else { \n"
							body_func += ff + "= " + "paramname_" + ff + "\n}\n"
						}

						if l.Enum != nil {
							if len(l.Default) > 0 {
								body_func += "if " +  ff + " == \"\"{\n"
								body_func += ff + " = " + "\"" +  l.Default + "\"\n}\n"
							}
							body_func += " m := make(map[string]bool)\n"
							for _, k := range l.Enum {
								body_func += "m[\"" + k + "\"] = true\n"
							}
							body_func += "_, prs := m[" + ff + "]\n"
							body_func += "if prs == false {\n"
							body_func += "js, _ := json.Marshal(JsonError{\"" + ff + " must be one of ["
							enums := ""
							for _, k := range l.Enum {
								enums += k + ", "
							}
							body_func += strings.TrimRight(enums, " ,")
							body_func += "]\"})"
							body_func += "\n"
							body_func += `w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}`
							body_func += "\n"
						}
						if len(l.Min) > 0 {
							if l.IsInt {
								body_func += " if !(" + ff +" >= " + l.Min + ") { \n"
								body_func += "js, _ := json.Marshal(JsonError{\"" + ff +  " must be >= " + l.Min + "\"})\n"
								body_func += `w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return`
								body_func += "}\n"
							} else {
								body_func += " if !(len(" + ff +") >= " + l.Min + ") { \n"
								body_func += "js, _ := json.Marshal(JsonError{\"" + ff +  " len must be >= " + l.Min + "\"})\n"
								body_func += `w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return`
								body_func += "}\n"
							}
						}
						if len(l.Max) > 0 {
							if l.IsInt {
								body_func += " if !(" + ff +" <= " + l.Max + ") { \n"
								body_func += "js, _ := json.Marshal(JsonError{\"" + ff +  " must be <= " + l.Max + "\"})\n"
								body_func += `w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return`
								body_func += "}\n"
							} else {
								body_func += " if !(len(" + ff +") <= " + l.Max + ") { \n"
								body_func += "js, _ := json.Marshal(JsonError{\"" + ff +  " must be <= " + l.Max + "\"})\n"
								body_func += `w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return`
								body_func += "}\n"
							}
						}
					}

				}

				body_func += j.In + " := " + j.In + " { "
				for s, _ := range params {
					body_func += strings.TrimRight(s, ",") + ": " + strings.ToLower(s)
				}
				body_func += " }"
				body_func += "\n"
				body_func += strings.ToLower(j.Out) + ", err := h." + j.Handler + " (ctx, " + j.In + ")"
				body_func += "\n"
				body_func += `if err != nil {
		switch err.(type) {
		case ApiError:
			js, _ := json.Marshal(JsonError{err.(ApiError).Err.Error()})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(err.(ApiError).HTTPStatus)
			w.Write(js)
			return
		default:
			js, _ := json.Marshal(JsonError{"bad user"})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(js)
			return
		}
	}`
				body_func += "\n"
				body_func += "js, _ := json.Marshal(Response" + key + j.Handler + " {" + strings.ToLower(j.Out) + ", JsonError{\"\"}})\n"
				body_func += `w.Header().Set("Content-Type", "application/json")
	w.Write(js)`
			}
		}
	}
	body_func += "\n}\n"
	fmt.Fprintln(out, body_func)

}
