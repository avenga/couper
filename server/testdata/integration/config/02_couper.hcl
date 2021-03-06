server "backends" {
  api {
    endpoint "/anything" {
      add_query_params = {
        bar = "3"
      }
      proxy {
        backend {
          origin = env.COUPER_TEST_BACKEND_ADDR
          add_response_headers = {
            foo = "4"
          }
          add_query_params = {
            bar = "4"
          }
        }
      }
    }

    endpoint "/" {
      proxy {
        backend "b" {
          add_response_headers = {
            foo = "4"
          }
        }
      }
    }

    endpoint "/get" {
      add_query_params = {
        bar = "3"
      }
      proxy {
        backend "a" {
          add_response_headers = {
            foo = "3"
          }
          add_query_params = {
            bar = "4"
          }
        }
      }
    }
  }
}

definitions {
  backend "b" {
    origin = env.COUPER_TEST_BACKEND_ADDR
    set_response_headers = {
      foo = "1"
    }
    set_query_params = {
      bar = "1"
    }
  }
  backend "a" {
    origin = env.COUPER_TEST_BACKEND_ADDR
    path = "/anything"
    set_response_headers = {
      foo = "1"
    }
    set_query_params = {
      bar = "1"
    }
  }
}

settings {
  no_proxy_from_env = true
}
