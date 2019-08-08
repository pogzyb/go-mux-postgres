package pages

type Page struct {
	Title		string
	Filename	string
	Alert		[2]string
	Data		map[string]interface{}
}