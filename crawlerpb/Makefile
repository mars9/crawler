crawler.pb.go: crawler.proto
	mkdir -p _pb
	protoc --go_out=_pb $<
	mv _pb/$@ $@ && rm -rf _pb
