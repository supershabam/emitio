.PHONY: deploy artifacts artifacts/emitio_linux_amd64 artifacts/edge_linux_amd64

deploy: artifacts
	ansible-playbook site.yml -i inventories/staging

artifacts: artifacts/emitio_linux_amd64 artifacts/edge_linux_amd64

artifacts/emitio_linux_amd64:
	make -C ../emitio bin
	cp ../emitio/target/emitio_linux_amd64 $@

artifacts/edge_linux_amd64:
	make -C ../edge target/edge_linux_amd64
	cp ../edge/target/edge_linux_amd64 $@
