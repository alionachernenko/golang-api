package keys

import "os"

var JWT_SECRET_KEY = []byte(os.Getenv("JWT_SECRET_KEY"))
var BUCKET_NAME = os.Getenv("BUCKET_NAME")
var AWS_REGION = os.Getenv("AWS_REGION")
var AWS_ACCESS_KEY = os.Getenv("AWS_ACCESS_KEY")
var AWS_SECRET_KEY = os.Getenv("AWS_SECRET_KEY")
