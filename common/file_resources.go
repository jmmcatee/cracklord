package common

/*
	FileResources are a map of FileResource so multiple different file key,
	value pairs
*/
type FileResources map[string]FileResource

/*
	FileResource is a key, value pair where the key is the usable name of the
	resource and the value is the full network path.
*/
type FileResource map[string]string
