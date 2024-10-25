package main

func getForm(f Getter, key string) ([]byte, error) {
	// .... 不能找到key
	return f.Get(key)
}

func test(key string) ([]byte, error) {
	return []byte(key), nil
}

type DB struct {
	url string
}

func (db *DB) Get(key string) ([]byte, error) {
	// ... 从数据库中获取数据
	return []byte(key), nil
}

func main() {
	getForm(new(DB), "abc")          // 将直接实现了Getter接口的结构体传入
	getForm(GetterFunc(test), "abc") // 将函数强制转换为GetterFunc，而GetterFunc实现了Getter的接口
}
