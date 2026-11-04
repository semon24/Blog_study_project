$ErrorActionPreference = 'Stop'

Push-Location (Join-Path $PSScriptRoot '..')
try {
    $gatewayModuleDir = (& go list -m -f '{{.Dir}}' github.com/grpc-ecosystem/grpc-gateway/v2).Trim()
    if ($LASTEXITCODE -ne 0) { throw 'Cannot find the grpc-gateway module. Run go mod download.' }

    $protocIncludeDir = Join-Path (Split-Path (Split-Path (Get-Command protoc).Source)) 'include'
    New-Item -ItemType Directory -Path api/openapi -Force | Out-Null

    & protoc -I . -I third_party -I $protocIncludeDir -I $gatewayModuleDir `
        --go_out=. --go_opt=module=blog-study-project `
        --go-grpc_out=. --go-grpc_opt=module=blog-study-project `
        --grpc-gateway_out=. --grpc-gateway_opt=module=blog-study-project `
        --openapiv2_out=api/openapi `
        --openapiv2_opt=allow_merge=true,merge_file_name=blog `
        api/blog/v1/blog.proto
    if ($LASTEXITCODE -ne 0) { throw 'Code generation failed.' }
}
finally {
    Pop-Location
}
