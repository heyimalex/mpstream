package mpstream

/*

RFC: https://tools.ietf.org/html/rfc7578

part


Parts
	Must have content-disposition form-data, name=""
	May also have filename
	May also have content-type header, defaults to text-plain, files should be guessed and/or labeled with application/octet-stream
	No other mime headers should be sent


*/
