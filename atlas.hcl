data "external_schema" "gorm" {
  program = [
    "go", "run", "-mod=mod",
    "ariga.io/atlas-provider-gorm",
    "load",
    "--path", "./internal/data/model",
    "--dialect", "mysql",
  ]
}

env "local" {
  src = data.external_schema.gorm.url
  dev = "docker://mysql/8/dev"
  migration {
    dir = "file://scripts/sql/migration"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}
