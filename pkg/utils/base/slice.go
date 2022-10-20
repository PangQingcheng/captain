package base

// HasString ... judging slice has such str or not
func HasString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// true有重复的元素，false无相同元素
func IsSameSlice(str1 []string, str2 []string) (t bool) {
	t = false
	if (str1 == nil) != (str2 == nil) {
		return
	}
	if len(str1) == 0 || len(str2) == 0 {
		return
	}
	map1, map2 := make(map[string]int), make(map[string]int)
	for i := 0; i < len(str1); i++ {
		map1[str1[i]] = i
	}
	for i := 0; i < len(str2); i++ {
		map2[str2[i]] = i
	}
	for k, _ := range map1 {
		if _, ok := map2[k]; !ok {
			return
		}
	}
	return true
}
