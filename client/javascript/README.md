### Instructions

Assuming your `$PWD` is `client/javascript`, and the gRPC server is running,
execute the following to create an HTTP server at `:8000` that serves the `./dist`
directory

```bash
# Python3
$ python -m http.server --directory dist
```

If you have Python 2, please use the following instead

```bash
# Python2
$ cd dist
$ python -m SimpleHTTPServer
```

Navigate to http://localhost:8000 and check the console for the `CountResponse`