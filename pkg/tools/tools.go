package tools


/* 一个通用的包含方案
func contains(slice interface{}, element interface{}) (bool, error) {
	value := reflect.ValueOf(slice)

	// 确保切片参数是一个切片。
	// 来自: https://github.com/golang/go/wiki/Kind
	if value.Kind() != reflect.Slice {
		return false, fmt.Errorf("slice parameter is not a slice: %+v", slice)
	}

	for i := 0; i < value.Len(); i++ {
		if reflect.DeepEqual(value.Index(i).Interface(), element) {
			return true, nil
		}
	}
	return false, nil
}
*/


// 判断一个元素是否包含在一个分片中
func Contains(slice []string, element string) bool {
	for _, value := range slice {
		if value == element {
			return true
		}
	}
	return false
}
