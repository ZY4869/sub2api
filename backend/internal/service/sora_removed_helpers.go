package service

func firstMediaURL(urls []string) string {
	for _, url := range urls {
		if url != "" {
			return url
		}
	}
	return ""
}
