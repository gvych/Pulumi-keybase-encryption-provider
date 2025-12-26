module decrypt-example

go 1.24.11

replace github.com/pulumi/pulumi-keybase-encryption => ../..

require (
	github.com/keybase/saltpack v0.0.0-20251212154201-989135827042
	github.com/pulumi/pulumi-keybase-encryption v0.0.0-00010101000000-000000000000
)

require (
	github.com/googleapis/gax-go/v2 v2.15.0 // indirect
	github.com/keybase/go-codec v0.0.0-20180928230036-164397562123 // indirect
	gocloud.dev v0.44.0 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	google.golang.org/api v0.247.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250811230008-5f3141c8851a // indirect
	google.golang.org/grpc v1.74.2 // indirect
	google.golang.org/protobuf v1.36.7 // indirect
)
