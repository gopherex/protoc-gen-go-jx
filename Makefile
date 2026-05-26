
.PHONY: gen-opts
gen-opts:
	rm -rf example/golden/*.go && easyp generate
