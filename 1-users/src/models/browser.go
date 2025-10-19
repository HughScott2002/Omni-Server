package models

type BrowserPattern struct {
	Name    string
	Pattern string
}

var BrowserPatterns = []BrowserPattern{
	// Major Desktop Browsers
	{Name: "Chrome", Pattern: "Chrome"},
	{Name: "Firefox", Pattern: "Firefox"},
	{Name: "Safari", Pattern: "Safari"},
	{Name: "Edge", Pattern: "Edge"},
	{Name: "Opera", Pattern: "OPR"},
	{Name: "Internet Explorer", Pattern: "MSIE"},
	{Name: "Brave", Pattern: "Brave"},
	{Name: "Vivaldi", Pattern: "Vivaldi"},
	{Name: "Zen", Pattern: "Zen"},
	{Name: "Arc", Pattern: "Arc"},
	{Name: "LadyBird", Pattern: "LadyBird"},

	// Mobile Browsers
	{Name: "Chrome Mobile", Pattern: "Chrome Mobile"},
	{Name: "Firefox Mobile", Pattern: "Firefox Mobile"},
	{Name: "Safari Mobile", Pattern: "Mobile Safari"},
	{Name: "Opera Mobile", Pattern: "Opera Mobile"},
	{Name: "Edge Mobile", Pattern: "EdgA"},
	{Name: "Samsung Browser", Pattern: "SamsungBrowser"},
	{Name: "UC Browser", Pattern: "UCBrowser"},

	// Development Tools
	{Name: "Postman", Pattern: "Postman"},
	{Name: "cURL", Pattern: "curl"},
	{Name: "wget", Pattern: "Wget"},
	{Name: "Insomnia", Pattern: "Insomnia"},
}
