# Setup
Create a config.json file in the format:
```json
{
  "/url/path": {
    "cmd": "echo 'command to be run with arg1 - %s & arg2 - %s'",
    "args": [
      "arg1", "arg2"
    ],
    "token": "*20 char min token*"
  }
}
```

You will then be able to have the `cmd` ran on the host via a http request:
```bash
$ curl http://127.0.0.1:8080/url/path -d 'token=*20 char min token*' -d 'arg1=foo' -d 'arg1=foo'
'command to be run with arg1 - foo & arg2 - bar'
```