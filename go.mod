module github.com/995933447/easytask

go 1.19

require (
	github.com/995933447/autoelect v0.0.0-20221205105211-eff1e99f1cfb
	github.com/995933447/confloader v0.0.0-20221201090521-fc5ce6a68064
	github.com/995933447/dbdriverutil v0.0.0-20221221100824-f994880a6ff1
	github.com/995933447/distribmu v0.0.0-20221129161736-69f07b237cf4
	github.com/995933447/log-go v0.0.0-20221218090749-3357926f7bc6
	github.com/995933447/optionstream v0.0.0-20220816081607-a125989b4cd9
	github.com/995933447/redisgroup v0.0.0-20220803160352-e08d00f81719
	github.com/995933447/reflectutil v0.0.0-20220816152525-eaa34e263589
	github.com/995933447/simpletrace v0.0.0-20221215132514-4140c70e8f71
	github.com/995933447/std-go v0.0.0-20220806175833-ab3496c0b696
	github.com/etcd-io/etcd v3.3.27+incompatible
	github.com/ggicci/httpin v0.10.1
	github.com/go-playground/validator v9.31.0+incompatible
	github.com/gorhill/cronexpr v0.0.0-20180427100037-88b0669f7d75
	gorm.io/driver/mysql v1.4.4
	gorm.io/gorm v1.24.2
	gorm.io/plugin/soft_delete v1.2.0
)

require (
	github.com/995933447/stringhelper-go v0.0.0-20221220072216-628db3bc29d8 // indirect
	github.com/BurntSushi/toml v1.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/coreos/etcd v3.3.27+incompatible // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-playground/locales v0.14.0 // indirect
	github.com/go-playground/universal-translator v0.18.0 // indirect
	github.com/go-redis/redis/v8 v8.11.5 // indirect
	github.com/go-sql-driver/mysql v1.6.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
)

replace (
	github.com/coreos/bbolt => go.etcd.io/bbolt v1.3.5
	google.golang.org/grpc => google.golang.org/grpc v1.26.0
)
