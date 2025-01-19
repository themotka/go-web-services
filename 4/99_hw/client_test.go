package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

type Root struct {
	XMLName xml.Name `xml:"root"`
	Row     []struct {
		ID        int    `xml:"id" json:"id"`
		Age       int    `xml:"age" json:"age"`
		FirstName string `xml:"first_name"`
		LastName  string `xml:"last_name"`
		Gender    string `xml:"gender" json:"gender"`
		About     string `xml:"about" json:"about"`
		Name      string `json:"name"`
	} `xml:"row"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("AccessToken")
	if token != "3" {
		w.WriteHeader(http.StatusUnauthorized)
	}
	query := r.URL.Query().Get("query")
	orderField := r.URL.Query().Get("order_field")
	orderBy, _ := strconv.Atoi(r.URL.Query().Get("order_by"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	menu := new(Root)
	xmlFile, _ := os.Open("dataset.xml")
	f := func(xmlFile *os.File) {
		err := xmlFile.Close()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	defer f(xmlFile)
	byteValue, _ := io.ReadAll(xmlFile)
	err := xml.Unmarshal(byteValue, &menu)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	for i, row := range menu.Row {
		menu.Row[i].Name = row.FirstName + " " + row.LastName
	}
	if orderField != "" {
		for i := len(menu.Row) - 1; i >= 0; i-- {
			row := menu.Row[i]
			row.Name = row.FirstName + " " + row.LastName
			if !strings.Contains(row.Name, query) && !strings.Contains(row.About, query) {
				menu.Row = append(menu.Row[:i], menu.Row[i+1:]...)
			}
		}
	}
	if orderField != "" && orderField != "Id" && orderField != "Age" && orderField != "Name" {
		w.WriteHeader(http.StatusBadRequest)
		marshal, _ := json.Marshal(ErrorResponse{Error: "ErrorBadOrderField"})
		w.Write(marshal)
		return
	}
	if orderBy != OrderByAsIs && orderBy != OrderByDesc && orderBy != OrderByAsc {
		w.WriteHeader(http.StatusBadRequest)
		marshal, _ := json.Marshal(ErrorResponse{Error: "Wrong OrderBy"})
		w.Write(marshal)
		return
	}
	if orderBy != OrderByAsIs {
		sort.Slice(menu.Row, func(i, j int) bool {
			if orderBy == OrderByDesc {
				switch orderField {
				case "Id":
					return menu.Row[i].ID < menu.Row[j].ID
				case "Age":
					return menu.Row[i].Age < menu.Row[j].Age
				case "Name":
					fallthrough
				case "":
					return menu.Row[i].Name < menu.Row[j].Name
				default:
					w.WriteHeader(http.StatusBadRequest)
					marshal, _ := json.Marshal(ErrorResponse{Error: fmt.Sprintf("Unknown field %s", orderField)})
					w.Write(marshal)
				}
			} else {
				switch orderField {
				case "Id":
					return menu.Row[i].ID > menu.Row[j].ID
				case "Age":
					return menu.Row[i].Age > menu.Row[j].Age
				case "Name":
					fallthrough
				case "":
					return menu.Row[i].Name > menu.Row[j].Name
				}
			}
			return true
		})
	}
	if limit > len(menu.Row) {
		limit = len(menu.Row)
	}
	if limit+offset-1 > len(menu.Row) {
		offset = len(menu.Row) - limit + 1
	}
	resp, _ := json.Marshal(menu.Row[len(menu.Row)-limit:])
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

type Test struct {
	Name    string
	Token   string
	Request SearchRequest
	IsError bool
	Error   string
}

func TestFindUsersError(t *testing.T) {
	cases := []Test{
		{
			"ok",
			"3",
			SearchRequest{
				Limit:      26,
				Offset:     0,
				Query:      "",
				OrderField: "",
				OrderBy:    OrderByAsc,
			},
			false,
			"",
		},
		{
			"bad token",
			"2",
			SearchRequest{
				Limit:      1,
				Offset:     1,
				Query:      "",
				OrderField: "",
				OrderBy:    OrderByAsIs,
			},
			true,
			"Bad AccessToken",
		},
		{
			"bad OrderField",
			"3",
			SearchRequest{
				Limit:      3,
				Offset:     0,
				Query:      "",
				OrderField: "bad",
				OrderBy:    OrderByDesc,
			},
			true,
			"OrderFeld bad invalid",
		},
		{
			"bad request",
			"3",
			SearchRequest{
				Limit:      3,
				Offset:     0,
				Query:      "",
				OrderField: "",
				OrderBy:    -2,
			},
			true,
			"unknown bad request error: Wrong OrderBy",
		},
		{
			"bad limit",
			"3",
			SearchRequest{
				Limit:      -2,
				Offset:     0,
				Query:      "",
				OrderField: "",
				OrderBy:    OrderByAsc,
			},
			true,
			"limit must be > 0",
		},
		{
			"bad offset",
			"3",
			SearchRequest{
				Limit:      3,
				Offset:     -1,
				Query:      "",
				OrderField: "",
				OrderBy:    OrderByAsc,
			},
			true,
			"offset must be > 0",
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			cl := SearchClient{
				URL:         ts.URL,
				AccessToken: tt.Token,
			}
			_, err := cl.FindUsers(tt.Request)
			if tt.IsError {
				assert.Equal(t, tt.Error, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFindUsersTimeout(t *testing.T) {
	tc := Test{
		"ok",
		"3",
		SearchRequest{
			Limit:      26,
			Offset:     0,
			Query:      "",
			OrderField: "",
			OrderBy:    OrderByAsc,
		},
		true,
		"unknown error Get \"?limit=26&offset=0&order_by=-1&order_field=&query=\": unsupported protocol scheme \"\"",
	}
	t.Run(tc.Name, func(t *testing.T) {
		cl := SearchClient{
			URL:         "",
			AccessToken: tc.Token,
		}
		resp, err := cl.FindUsers(tc.Request)
		fmt.Println(resp)
		if tc.IsError {
			assert.Equal(t, tc.Error, err.Error())
		} else {
			assert.NoError(t, err)
		}
	})
}
