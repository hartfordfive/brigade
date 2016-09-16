package main

import (
	"encoding/json"
	"fmt"
	j "github.com/gima/jsonv/src"
	"os"
	"strconv"
	"time"
)

func ValidateHttpDirective(json_data string) (interface{}, error) {

	/*
		schema := &j.Object{Properties: []j.ObjectItem{
			{"status", &j.Boolean{}},
			{"data", &j.Object{Properties: []j.ObjectItem{
				{"token", &j.Function{myValidatorFunc}},
				{"debug", &j.Number{Min: 1, Max: 99999}},
				{"items", &j.Array{Each: &j.Object{Properties: []j.ObjectItem{
					{"url", &j.String{MinLen: 1}},
					{"comment", &j.Optional{&j.String{}}},
				}}}},
				{"ghost", &j.Optional{&j.String{}}},
				{"ghost2", &j.Optional{&j.String{}}},
				{"meta", &j.Object{Each: j.ObjectEach{
					&j.String{}, &j.Or{&j.Number{Min: .01, Max: 1.1}, &j.String{}},
				}}},
			}}},
		}}
	*/

	schema := &j.Object{
		Properties: []j.ObjectItem{
			{"type", &j.String{MinLen: 3}},
			{"url_list", &j.Array{Each: &j.Object{Properties: []j.ObjectItem{
				{"on_startup_only", &j.Optional{&j.Boolean{}}},
				{"url", &j.String{MinLen: 7}},
				{"method", &j.String{MinLen: 3, MaxLen: 4}},
				{"cookies", &j.Optional{&j.Array{Each: &j.Object{Properties: []j.ObjectItem{
					{"name", &j.String{MinLen: 1}},
					{"value", &j.Optional{&j.String{}}},
				}}}}},
				{"post_data", &j.Optional{&j.Array{Each: &j.Object{Properties: []j.ObjectItem{
					{"name", &j.String{MinLen: 1}},
					{"value", &j.Optional{&j.String{}}},
				}}}}},
				{"weight", &j.Number{Min: 0.1}},
			}}}},
			{"proxy", &j.Optional{&j.String{}}},
			{"min_delay", &j.Number{Min: 0}},
			{"max_delay", &j.Number{Min: 0}},
		},
	}

	decoded := new(interface{})
	if err := json.Unmarshal([]byte(json_data), decoded); err != nil {
		//log.Fatal("[ERROR] JSON parsing failed:", err)
		return decoded, err
	}

	if _, err := schema.Validate(decoded); err != nil {
		return decoded, err
	}

	return decoded, nil
}

func ValidateScriptDirective(json_data string) (interface{}, error) {

	schema := &j.Object{
		Properties: []j.ObjectItem{
			{"type", &j.String{MinLen: 3}},
			{"script_name", &j.String{}},
			{"script_type", &j.String{}},
			{"script_body", &j.Optional{&j.String{}}},
			{"script_url", &j.Optional{&j.String{MinLen: 11}}},
			{"repeat_mode", &j.Optional{&j.String{}}},
			{"repeat_times", &j.Optional{&j.Number{Min: 0}}},
		},
	}

	decoded := new(interface{})
	if err := json.Unmarshal([]byte(json_data), decoded); err != nil {
		//log.Fatal("[ERROR] JSON parsing failed:", err)
		return decoded, err
	}

	if _, err := schema.Validate(decoded); err != nil {
		return decoded, err
	}

	return decoded, nil
}

func YmdToString() string {
	t := time.Now()
	y, m, d := t.Date()
	return strconv.Itoa(y) + "-" + fmt.Sprintf("%02d", m) + "-" + fmt.Sprintf("%02d", d)
}

func YmdAndTimeToString() string {
	t := time.Now()
	y, m, d := t.Date()
	return strconv.Itoa(y) + "-" + fmt.Sprintf("%02d", m) + "-" + fmt.Sprintf("%02d", d) + "-" + fmt.Sprintf("%02d", t.Hour()) + fmt.Sprintf("%02d", t.Minute()) + fmt.Sprintf("%02d", t.Second())
}

func DateStampAsString() string {
	t := time.Now()
	return YmdToString() + " " + fmt.Sprintf("%02d", t.Hour()) + ":" + fmt.Sprintf("%02d", t.Minute()) + ":" + fmt.Sprintf("%02d", t.Second())
}

func PersistDirectivesToDisk(data map[string]interface{}, file_path string) (bool, error) {

	f, err := os.Create(file_path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	json_bytes, _ := json.Marshal(data)
	_, err = f.Write(json_bytes)
	if err != nil {
		return false, err
	}
	f.Sync()

	return true, nil
}

func PrintDebug(lvl int, str string) {

	switch lvl {
	case 0:
		fmt.Printf("[INFO] %s :: %s", DateStampAsString(), str)
	case 1:
		fmt.Printf("[WARNING] %s :: %s", DateStampAsString(), str)
	case 2:
		fmt.Printf("[ERROR] %s :: %s", DateStampAsString(), str)
	case 3:
		fmt.Printf("[DEBUG] %s :: %s", DateStampAsString(), str)
	}
}
