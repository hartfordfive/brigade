package main

import (
	"fmt"
	"math/rand"
	//"reflect"
	"strconv"
	//"strings"
	crand "crypto/rand"
	"math/big"
	"time"
)

var CHAR_SET = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890~!@#$%^&*()_-=+")
var CHAR_SET_ALPHA_NUM = CHAR_SET[0:35]

var used_passwords = map[string]bool{}

func GetMonthAsIntString(m string) string {

	switch m {
	case "January":
		return "01"
	case "Februrary":
		return "02"
	case "March":
		return "03"
	case "April":
		return "04"
	case "May":
		return "05"
	case "June":
		return "06"
	case "July":
		return "07"
	case "August":
		return "08"
	case "September":
		return "09"
	case "October":
		return "10"
	case "November":
		return "11"
	case "December":
		return "12"
	}
	return "01"
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

func GetRndPass(size int) string {

	b := make([]rune, size)
	for i := range b {
		b[i] = CHAR_SET[rand.Intn(len(CHAR_SET))]
	}
	return string(b)

}

func GetRndStr(n int) string {

	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	symbols := big.NewInt(int64(len(alphanum)))
	states := big.NewInt(0)
	states.Exp(symbols, big.NewInt(int64(n)), nil)
	r, err := crand.Int(crand.Reader, states)
	if err != nil {
		panic(err)
	}
	var bytes = make([]byte, n)
	r2 := big.NewInt(0)
	symbol := big.NewInt(0)
	for i := range bytes {
		r2.DivMod(r, symbols, symbol)
		r, r2 = r2, r
		bytes[i] = alphanum[symbol.Int64()]
	}
	return string(bytes)

}

func WeightToPercentage(weight int, total int) int {
	return int((weight * 100) / total)
}

func CalculateWeightPercentageMap(weights []int) map[int]int {

	var total_weight int
	var weight_percentages map[int]int

	for i := 0; i < len(weights); i++ {
		total_weight += weights[i]
	}

	for i := 0; i < len(weights); i++ {
		weight_percentages[weights[i]] = WeightToPercentage(weights[i], total_weight)
	}

	return weight_percentages
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

/*
func GetDirectivesTotalWeight(directives []interface{}) int {
	//fmt.Println(reflect.New(reflect.TypeOf(obj)).Interface())
	var total_weight int
	for _, d := range directives {
		r := reflect.ValueOf(d)
		//f := int(reflect.Indirect(r).FieldByName("Weight"))
		total_weight += int(reflect.Indirect(r).FieldByName("Weight").Int())
	}
	return int(total_weight)
}

func DirectiveInProbability(d interface{}, total_directives int, total_weight int) bool {

	var dnum, nw, probability int
	var t string

	rt := reflect.TypeOf(d)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
		parts := strings.Split(rt.String(), ".")
		t = parts[1]
	}

	r := reflect.ValueOf(d)
	f := reflect.Indirect(r).FieldByName("Weight")

	dnum = rand.Intn(total_directives)
	nw = WeightToPercentage(int(f.Int()), total_weight)
	probability = rand.Intn(100) + 1
	if probability >= nw {
		return false
	}

	return true
}
*/
