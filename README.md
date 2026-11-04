# no-plans

NO PLANS — Stop scrolling. Start going.

Ứng dụng giúp nhóm bạn tạo kèo, mời thành viên, xác nhận tham gia, bình chọn địa điểm và chốt cuộc hẹn.

## Trạng thái

MVP đã có ứng dụng Expo, web lời mời, API Go và worker. Các luồng chính gồm tạo kèo, lời mời, RSVP, bình chọn địa điểm, nhắc lịch, báo cáo và xóa hồ sơ. Xem phần xác minh trên staging để biết các bước còn lại trước beta.

- [Thiết kế hệ thống](docs/system-design.md)
- [Ngữ cảnh và tiến độ](codex.md)
- [Kế hoạch triển khai](docs/superpowers/plans/2026-11-01-mvp.md)
- [Luồng và wireframe](docs/ux/core-flows.md)
- [API contract và cách gọi](docs/api.md)

## Backend local

Yêu cầu: Go 1.25+, PostgreSQL 15+ hoặc Docker Engine. Compose dùng PostgreSQL 17, bind cổng local 54322.

```powershell
docker compose up -d postgres
$env:DATABASE_URL='postgres://no_plans:no_plans_local@localhost:54322/no_plans?sslmode=disable'
$env:AUTH_ISSUER='http://localhost:54321/auth/v1'
$env:AUTH_AUDIENCE='authenticated'
$env:AUTH_API_KEY='YOUR_SUPABASE_PUBLISHABLE_KEY'
$env:CORS_ORIGINS='http://localhost:8081,http://localhost:5173'
cd backend
go run ./cmd/migrate
go run ./cmd/api
```

`.env.example` là mẫu tham khảo; backend đọc environment của tiến trình, không tự nạp file `.env`. Auth issuer phải trỏ đến Supabase Auth dùng asymmetric signing keys (ES256/RS256); chưa cấu hình provider thực. Migration chạy riêng, không tự chạy khi API khởi động. Production cần role API hạn chế quyền, migration role riêng, HTTPS issuer và database `sslmode=verify-full`.

`GET /healthz` kiểm tra process; `GET /readyz` kiểm tra database với deadline 2 giây. Các probes public. Các API `/v1` được nối vào binary, yêu cầu JWT thật từ issuer đã cấu hình; xem [API contract](docs/api.md). Trước khi gửi lời mời, provider phải xác nhận email đã được xác minh.

## Kiểm thử

```powershell
cd backend
go test ./...
go vet ./...
go build -o bin/api.exe ./cmd/api
```

Integration test cần **database test riêng, có thể xóa dữ liệu**, role được tạo schema; không trỏ tới production. Mỗi test tạo schema ngẫu nhiên rồi dọn schema của chính nó, cô lập quota, trigger lỗi và dữ liệu fixture.

```powershell
$env:TEST_DATABASE_URL='postgres://USER:PASSWORD@localhost:PORT/no_plans_test?sslmode=disable'
go test ./... -count=1 -v
```

Không có TEST_DATABASE_URL thì PostgreSQL test bị skip. Auth test tạo HTTP/JWKS server loopback và ký token thật, không dùng Supabase cloud. Nếu sandbox chặn loopback, chạy test với quyền network local phù hợp.

## Phạm vi chưa hoàn tất

Mobile Expo, web lời mời, worker push, báo cáo và xóa hồ sơ đã có triển khai. Beta còn cần xác minh Supabase cloud, thông báo trên thiết bị thật và kiểm thử PostgreSQL 17 trên CI.

## Invite links và catalogue

Backend cần `INVITE_TOKEN_SECRET` là base64 của32 byte ngẫu nhiên để tạo link; production bắt buộc cấu hình. Không commit secret. Database chỉ lưu SHA256 token; idempotent replay tái tạo token bằng secret. Đổi secret không vô hiệu link đã phát hành nhưng replay key cũ sẽ nhận409; tạo request với idempotency key mới khi cần link mới.

Sau migration, chạy `go run ./cmd/seed` trong backend với DATABASE_URL để nạp catalogue ban đầu, hoặc thêm `-file path/to/catalogue.json`. Lệnh dùng quyền operator, không có HTTP endpoint sửa catalogue. Catalogue phân biệt địa điểm có nguồn và ý tưởng hoạt động; chi phí ý tưởng là dự trù, giá địa điểm chưa biết để null.

## Web invite local

Yêu cầu Node>=22.12. Trong `apps/invite`, chạy `npm ci`, sao chép `.env.example` thành `.env.local`, điền Supabase URL/publishable key rồi `npm run dev`. Backend chạy cổng8080; Vite proxy `/v1` đến backend. Supabase cần cho phép redirect về origin web. Không đưa secret backend vào biến VITE.

Link có dạng `http://localhost:5173/#token=TOKEN`; frontend bỏ fragment khỏi thanh địa chỉ, gửi token qua header và chỉ lưu tạm trong sessionStorage cho đến khi redeem thành công. Guest chỉ thấy tên, loại hoạt động và thời gian; địa điểm cần membership. Khi deploy, cấu hình reverse proxy `/v1` hoặc VITE_API_URL và exact CORS origin tương ứng, HTTPS và Referrer-Policy no-referrer.

Kiểm tra: `npm test`, `npm run build`; cài browser riêng bằng `npx playwright install chromium`, rồi `npm run test:e2e`. Browser suite dùng controlled HTTP fixtures cho Supabase/API; backend có integration tests riêng với PostgreSQL.

## Expo mobile (đang triển khai)

`apps/mobile` dùng Expo SDK58, React Native0.88 và Node>=24.3. Chạy `npm ci`, sao chép `.env.example` thành `.env.local` và điền API URL/Supabase public config, rồi `npm start`. Thiết bị thật cần API URL là IP LAN/HTTPS có thể truy cập từ thiết bị; localhost trên điện thoại không trỏ đến máy backend. Native không dùng CORS, bản web cần thêm exact origin vào backend CORS_ORIGINS.

Hiện có auth, danh sách kèo, tạo kèo, RSVP, hồ sơ và danh sách/tạo nhóm. Deep links, notifications, safety và các bước beta/device vẫn còn trong kế hoạch. Icons còn là template Expo. Các lệnh kiểm tra: `npm test`, `npm run typecheck`, `npm run lint`, `npm run export`. Export bundle không thay thế native build, kiểm thử UI hay beta verification.

Chủ kèo có thể sửa thông tin với If-Match, chốt, hủy hoặc kết thúc sau giờ hẹn. Xung đột version giữ nguyên bản sửa; người dùng chủ động bỏ bản sửa và tải snapshot mới để kiểm tra. Lỗi kết nối khi refresh giữ form hiện tại, còn lỗi quyền truy cập xóa chi tiết riêng tư.

Mobile có Bạn bè và chi tiết nhóm: gửi mã kết bạn, chấp nhận lời mời nhận được, owner thêm bạn đã kết nối, xóa thành viên và chuyển quyền; member có thể rời nhóm. Các thao tác xóa/chuyển/rời có xác nhận. Người rời nhóm vẫn giữ quyền kèo cũ; owner phải chuyển quyền trước khi rời. Luồng được kiểm tra trên Expo web với HTTP fixtures, còn cần kiểm chứng qua provider/backend thực và thiết bị native.

Chi tiết kèo có người tham gia/RSVP và quản lý lời mời: owner mời bạn, thu hồi membership, tạo link/chia sẻ và thu hồi link. Đặt `EXPO_PUBLIC_INVITE_WEB_URL` trỏ tới web invite đã deploy (HTTPS production; ví dụ local dev `http://YOUR_LAN_IP:5173`). Token chỉ nằm trong fragment khi chia sẻ và state màn hình, không lưu bền vững. Thu hồi link không xóa quyền của người đã tham gia; dùng thu hồi từng thành viên khi cần. Native share sheet còn cần kiểm chứng trên thiết bị thật.

Khám phá có bộ lọc hoạt động/loại/khu vực/ngân sách/số người và phân trang. Chi phí chưa biết không được hiển thị thành miễn phí. Trong kèo, chủ kèo đề xuất custom venue hoặc mục catalogue; mỗi thành viên một phiếu có thể đổi trước khi chốt. Ý tưởng vẫn nhận phiếu nhưng cần nơi gặp cụ thể để chốt. Migration006 lưu loại option vào snapshot; chọn địa điểm dùng If-Match và yêu cầu kiểm tra lại khi version thay đổi. Chọn nơi gặp không tự chốt kèo.

Trang lời mời có nút mở ứng dụng đã cài qua `no-plans://invite#token=…`. Người nhận cũng có thể tiếp tục trên web. App giữ lời mời qua đăng nhập và yêu cầu bấm tham gia trước khi redeem. Link riêng tư `no-plans://plans/<UUID>` mở màn kèo với kiểm tra quyền từ API. Kiểm thử thiết bị thật và cấu hình liên kết HTTPS tự mở app cần bản build cùng domain triển khai.

Form tạo kèo lưu nháp theo tài khoản trên thiết bị, giữ khóa retry qua reload và chỉ gửi sau khi lưu được khóa. Lần gửi không rõ kết quả quá23 giờ yêu cầu kiểm tra danh sách trước khi bỏ nháp để tránh gửi lại ngoài cửa sổ replay của server. Nháp chứa nội dung bạn nhập, không chứa JWT. `npm run test:e2e` kiểm tra Expo web với HTTP fixtures bằng Chromium riêng (`npx playwright install chromium`); chưa thay thế luồng native/Go/Supabase cloud thực.

### Worker

Sau khi chạy migration, khởi động tiến trình riêng trong thư mục backend.

```powershell
go run ./cmd/worker
```

Worker dùng cùng `DATABASE_URL` với API. Đặt `EXPO_ACCESS_TOKEN` nếu bật xác thực Expo Push. Cấu hình cả `SUPABASE_URL` và `SUPABASE_SERVICE_ROLE_KEY` trên worker để xử lý hàng đợi xóa danh tính. Giữ khóa quản trị trong cấu hình bí mật phía máy chủ.

### Diễn tập khôi phục PostgreSQL trước beta

Tạo backup hằng ngày bằng `pg_dump` ở định dạng custom. Lưu bản sao trong kho backup có mã hóa và giới hạn quyền đọc. Dùng thông tin kết nối được cung cấp cho operator qua cấu hình bí mật.

```powershell
pg_dump --dbname $env:DATABASE_URL --format=custom --file no-plans.dump
pg_restore --dbname $env:RESTORE_DATABASE_URL --no-owner --no-privileges --exit-on-error no-plans.dump
```

`RESTORE_DATABASE_URL` trỏ đến database trống dành riêng cho diễn tập. Sau restore, đối chiếu migration và số bản ghi của các bảng chính với bản sao, rồi chạy API trên database khôi phục để kiểm tra `/readyz` và đọc một kèo fixture. Ghi thời điểm backup, thời gian khôi phục và kết quả vào nhật ký vận hành. Chỉ chạy worker với dịch vụ ngoài giả lập trong môi trường diễn tập.

### Xác minh trên staging

- Chuẩn bị PostgreSQL 17, API và worker, một dự án Supabase Auth dành cho staging và hai tài khoản thử nghiệm có email đã xác minh.
- Liên kết ứng dụng với dự án EAS, cấu hình định danh ứng dụng và thông tin xác thực push cho iOS và Android. Trỏ biến môi trường công khai của mobile và web lời mời về staging.
- Trên hai thiết bị, tạo kèo, gửi lời mời, mở link khi ứng dụng đang chạy và khi đã đóng, rồi phản hồi và chốt kèo. Xác nhận thông báo đến đúng tài khoản và mở đúng kèo.
- Đổi giờ, hủy kèo và tắt thông báo nhóm để kiểm tra lượt gửi còn chờ. Dùng tài khoản thử nghiệm riêng để kiểm tra xóa hồ sơ, thu hồi push và hoàn tất xóa danh tính.
- Ghi phiên bản build, nền tảng, commit, kết quả và các bước tái hiện vào nhật ký kiểm thử.

### Theo dõi hàng đợi push trên staging

Chạy truy vấn chỉ đọc bằng tài khoản vận hành. Tuổi hàng đợi tính từ thời điểm sự kiện đến hạn, nên lời nhắc chưa đến giờ không làm tăng số liệu.

```sql
SELECT count(*) AS outbox_due,
       coalesce(max(extract(epoch FROM now() - available_at)), 0) AS oldest_due_seconds
FROM outbox
WHERE processed_at IS NULL AND dead_letter_at IS NULL AND available_at <= now();

SELECT count(*) AS deliveries_completed,
       count(*) FILTER (WHERE failure_code IS NOT NULL) AS deliveries_with_failure
FROM push_deliveries
WHERE completed_at >= now() - interval '1 hour';
```

Tỷ lệ lỗi tính trên các lượt đã hoàn tất còn trong bảng. Lượt gửi bị xóa theo thiết bị khi Expo trả `DeviceNotRegistered` không nằm trong số liệu này.

### Đo độ trễ HTTP trên staging

Xuất log JSON của API trong cửa sổ kiểm thử thành `api.jsonl`. Đo riêng các thao tác không gọi dịch vụ ngoài khi đối chiếu mục tiêu p95 500 ms. Trộn các đợt tải khác nhau sẽ làm kết quả khó so sánh.

```powershell
$requests = @(Get-Content api.jsonl | ForEach-Object {
    $entry = $_ | ConvertFrom-Json
    if ($entry.msg -eq 'http request') { $entry }
})
if ($requests.Count -eq 0) { throw 'No HTTP requests in sample' }
$durations = @($requests.duration_ms | Sort-Object)
[pscustomobject]@{
    Requests = $requests.Count
    P95Milliseconds = $durations[([int][Math]::Ceiling(0.95 * $durations.Count) - 1)]
    ServerErrorRate = @($requests | Where-Object { $_.status -ge 500 }).Count / $requests.Count
}
```
