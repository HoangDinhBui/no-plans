# no-plans

NO PLANS — Stop scrolling. Start going.

Ứng dụng giúp nhóm bạn tạo kèo, mời thành viên, xác nhận tham gia, bình chọn địa điểm và chốt cuộc hẹn.

## Trạng thái

Đang triển khai MVP theo thiết kế đã duyệt. Backend hiện có probes HTTP, quy tắc kèo, PostgreSQL migrations/persistence và xác minh JWT qua JWKS. Các routes nghiệp vụ và giao diện chưa hoàn tất.

- [Thiết kế hệ thống](docs/system-design.md)
- [Ngữ cảnh và tiến độ](codex.md)
- [Kế hoạch triển khai](docs/superpowers/plans/2026-11-01-mvp.md)
- [Luồng và wireframe](docs/ux/core-flows.md)

## Backend local

Yêu cầu: Go 1.25+, PostgreSQL 15+ hoặc Docker Engine. Compose dùng PostgreSQL 17, bind cổng local 54322.

```powershell
docker compose up -d postgres
$env:DATABASE_URL='postgres://no_plans:no_plans_local@localhost:54322/no_plans?sslmode=disable'
$env:AUTH_ISSUER='http://localhost:54321/auth/v1'
$env:AUTH_AUDIENCE='authenticated'
cd backend
go run ./cmd/migrate
go run ./cmd/api
```

`.env.example` là mẫu tham khảo; backend đọc environment của tiến trình, không tự nạp file `.env`. Auth issuer phải trỏ đến Supabase Auth dùng asymmetric signing keys (ES256/RS256); chưa cấu hình provider thực. Migration chạy riêng, không tự chạy khi API khởi động. Production cần role API hạn chế quyền, migration role riêng, HTTPS issuer và database `sslmode=verify-full`.

`GET /healthz` kiểm tra process; `GET /readyz` kiểm tra database với deadline 2 giây. Các probes public, không có dữ liệu người dùng. Chưa có API tạo kèo được expose; domain/persistence đang được hoàn thiện.

## Kiểm thử

```powershell
cd backend
go test ./...
go vet ./...
go build -o bin/api.exe ./cmd/api
```

Integration test cần **database test riêng, có thể xóa dữ liệu**; không trỏ tới production. Test tạo fixture cố định và xóa các kèo fixture.

```powershell
$env:TEST_DATABASE_URL='postgres://USER:PASSWORD@localhost:PORT/no_plans_test?sslmode=disable'
go test ./... -count=1 -v
```

Không có TEST_DATABASE_URL thì PostgreSQL test bị skip. Auth test tạo HTTP/JWKS server loopback và ký token thật, không dùng Supabase cloud. Nếu sandbox chặn loopback, chạy test với quyền network local phù hợp.

## Phạm vi chưa hoàn tất

Mobile Expo, web invite, nhóm/kết bạn, HTTP routes nghiệp vụ, voting/discovery, worker push và các bước beta/device/store vẫn đang trong kế hoạch; chưa sẵn sàng phát hành.
