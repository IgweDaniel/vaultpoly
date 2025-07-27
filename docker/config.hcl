ui            = true
api_addr      = "http://0.0.0.0:8200"

disable_mlock = false

storage "file" {
  path = "/vault/file/data"
}

listener "tcp" {
  address       = "0.0.0.0:8200"
  tls_disable = 1
}



plugin_directory = "/vault/plugins"