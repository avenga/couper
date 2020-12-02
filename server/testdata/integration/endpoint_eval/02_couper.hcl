server "api" {
  api {
    endpoint "/endpoint1" {
      path = "/anything"
      backend  = "anything"
    }

    #endpoint "/endpoint2" {
    #  path = "/unset/by/local-backend"
    #  backend "anything" {
    #    path = "/anything"
    #  }
    #}

    # don't override path
    endpoint "/endpoint3" {
      backend  = "anything"
    }
  }
}

definitions {
  # backend origin within a definition block gets replaced with the integration test "anything" server.
  backend "anything" {
    path = "/unset/by/endpoint"
    origin = "http://anyserver/"
  }
}
