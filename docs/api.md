# NO PLANS — API hiện có

Backend Go, prefix `/v1`, JSON. Routes `/v1` yêu cầu `Authorization: Bearer <Supabase access token>`, ngoại trừ `/v1/invite-preview` public; probes `/healthz`, `/readyz` public. Actor lấy từ JWT đã kiểm tra chữ ký, issuer, audience, expiry, role và subject UUID. Không có đường đăng nhập giả hay header tự chọn user.

## Xác thực và cấu hình

Mobile/web đăng ký, đăng nhập, xác minh email và refresh session bằng Supabase Auth. Go xác minh JWT qua `<AUTH_ISSUER>/.well-known/jwks.json`. Dùng signing keys ES256 hoặc RS256; legacy HS256 không được chấp nhận.

`AUTH_API_KEY` là publishable/anon API key của đúng Supabase project, dùng cùng access token để đọc `/user` và kiểm tra `email_confirmed_at` từ provider trước khi gửi lời mời. Không dùng service-role key. Nếu chưa cấu hình, routes cần xác minh email trả `503 auth_unavailable`; metadata `email_verified` do người dùng sửa không có quyền bỏ qua bước này.

Đăng nhập lần đầu tự tạo profile với tên “Bạn mới”. Lần đăng nhập sau giữ tên/avatar đã sửa. Tài khoản có `deleted_at` không được khôi phục tự động.

## Routes

| Method | Path | Body / mục đích |
|---|---|---|
| GET | `/me` | Hồ sơ của người gọi |
| PATCH | `/me` | `display_name`, `avatar_url` tùy chọn; trường không gửi được giữ nguyên |
| GET | `/friend-requests` | Các lời mời kết bạn pending/accepted của người gọi, kèm profile peer |
| POST | `/friend-requests` | `recipient_id`; cần email verified và Idempotency-Key |
| POST | `/friend-requests/{id}/accept` | Chỉ người nhận được chấp nhận; retry không tạo thêm sự kiện |
| GET | `/groups` | Nhóm mình đang tham gia |
| POST | `/groups` | `name`; cần Idempotency-Key; người tạo là owner và thành viên |
| GET | `/groups/{id}` | Thông tin và profile thành viên; chỉ thành viên được đọc |
| POST | `/groups/{id}/members/{userId}` | Owner thêm bạn đã kết bạn hai chiều; cần email verified |
| DELETE | `/groups/{id}/members/{userId}` | Owner xóa thành viên hoặc người gọi tự rời nhóm |
| PUT | `/groups/{id}/owner` | `user_id`; owner chuyển quyền cho thành viên hiện tại |
| GET | `/plans` | Kèo riêng tư của mình; bộ lọc `status` ở dưới |
| POST | `/plans` | Tạo kèo; cần email verified và Idempotency-Key |
| GET | `/plans/{id}` | Chi tiết, map user-ID → RSVP và ETag version |
| GET | `/plans/{id}/members` | Danh sách phân trang profile người tham gia + response; chỉ thành viên được xem |
| PATCH | `/plans/{id}` | Owner sửa trường gửi lên; bắt buộc If-Match |
| PUT | `/plans/{id}/rsvp` | `response`: going / not_going / maybe; chỉ sửa phản hồi của mình |
| POST | `/plans/{id}/confirm` | Owner chốt kèo có giờ tương lai và địa điểm |
| POST | `/plans/{id}/cancel` | Owner hủy kèo active |
| POST | `/plans/{id}/complete` | Owner kết thúc kèo đã chốt sau giờ bắt đầu |
| POST | `/plans/{id}/invitations` | `user_id`; owner mời thêm bạn đã kết bạn; email verified |
| DELETE | `/plans/{id}/invitations/{userId}` | Owner thu hồi membership; người bị thu hồi mất quyền truy cập |
| GET | `/blocks` | Profile người mình đã chặn |
| POST | `/blocks` | `user_id`; chặn lời mời mới cả hai chiều và xóa quan hệ kết bạn |
| DELETE | `/blocks/{userId}` | Bỏ chặn; không tự khôi phục kết bạn |

Owner nhóm phải chuyển quyền trước khi rời. Owner kèo tính vào sức chứa, nhưng có thể đổi RSVP như mọi thành viên. Việc rời/xóa khỏi nhóm không tự xóa membership kèo cũ. Thành viên nhóm được chụp lại tại lúc tạo kèo; người tham gia nhóm sau không tự có quyền đọc kèo cũ. Chặn không tự xóa membership nhóm/kèo hiện có; owner có thể thu hồi riêng.

## Tạo và sửa kèo

```json
{
  "title": "Tối nay cà phê",
  "activity": "coffee",
  "starts_at": "2026-12-01T19:30:00+07:00",
  "capacity": 6,
  "location": "Quán đã chọn",
  "group_id": "UUID-nhóm-tùy-chọn"
}
```

`activity`: food, coffee, drinks, cinema, sports, other. Title 1–120 ký tự Unicode; location tối đa 500. Capacity 0 là không giới hạn; nếu nhập giới hạn phải từ 1 đến 2147483647. `starts_at` phải trong tương lai. `group_id` bỏ trống nếu tạo kèo chỉ có chủ kèo, rồi mời bạn/link sau. Client không được tự gửi owner_id, members hoặc status.

GET trả `ETag: "3"`; PATCH gửi `If-Match: "3"`. Các trường PATCH tùy chọn: title, starts_at, capacity, location. Version cũ trả 409; không có If-Match trả 428. Không giảm capacity dưới số đang đi. Confirm/cancel/complete là commands riêng, không được sửa status qua PATCH.

Idempotency-Key dài 1–128 ký tự ASCII hiển thị; nên dùng UUID. Phạm vi key theo actor + loại tạo mới; giữ 24 giờ. Cùng key và nội dung đã chuẩn hóa trả cùng kết quả tạo ban đầu, kể cả khi retry đồng thời; khác nội dung trả 409. Replay không gửi lại thông báo hoặc tiêu thêm quota. Sau replay, tải lại GET để có trạng thái hiện tại nếu entity đã thay đổi.

## Danh sách và lỗi

Mọi list trả `{ "items": [], "next_cursor": "..." }`; không có trang kế tiếp thì bỏ next_cursor. `limit` mặc định 20, từ 1–100. Chỉ gửi lại cursor server trả; không tự tính offset.

Nhóm/bạn/chặn dùng cursor UUID và sắp theo ID tăng dần. Kèo sắp tới theo starts_at + ID tăng dần; lịch sử giảm dần. `/plans?status=upcoming` là mặc định; `invited` là lời mời chưa trả lời còn hiệu lực; `past` gồm kèo hủy/kết thúc hoặc đã qua giờ bắt đầu. Cursor kèo có gắn bộ lọc, không dùng lại giữa các status. Danh sách kèo có my_response, going_count và pending_count.

Lỗi JSON gồm `code` và `request_id`, đồng thời có header `X-Request-ID` để tra cứu. 400 input/cursor/JSON sai; 401 token không hợp lệ; 403 không có quyền hoặc email_unverified; 409 plan_full/plan_closed/version_conflict/idempotency_conflict; 413 body quá 64 KiB; 415 sai Content-Type; 428 thiếu version; 429 rate_limited; 503 auth_unavailable. Không trả lỗi SQL, token hoặc dữ liệu nhạy cảm trong response/log.

Quota beta hiện tại: tối đa 300 request authenticated/phút/actor; 200 người nhận lời mời/giờ/actor; 10 lời mời/ngày/cặp actor–recipient; 20 đợt lời mời/giờ/actor–group. Các giới hạn chia sẻ qua PostgreSQL; quota lời mời cùng transaction với nghiệp vụ và outbox. 429 có Retry-After 60 giây; giới hạn cửa sổ dài hơn có thể vẫn chưa hết sau lần thử đó. Worker retention cho dữ liệu quota sẽ hoàn thiện trước beta.

Web chỉ được gọi từ exact origin trong `CORS_ORIGINS`, ví dụ `http://localhost:5173`; không dùng wildcard hoặc URL có path. Preflight không cần JWT; credentials cookies không được bật. Native app không cần CORS origin. Cần cấu hình lại origins HTTPS khi production.

## Chưa có ở mốc này

Token invite links/deep link, voting/discovery, push worker, reports, account deletion và mobile/web UI đang được triển khai theo kế hoạch. Outbox đã lưu sự kiện nhưng chưa gửi push thật. Kiểm thử hiện dùng Supabase-compatible HTTP fixture và JWT ký thật; chưa chứng minh kết nối tài khoản Supabase cloud hay thiết bị thật.
## Invite links, voting và catalogue

Các endpoint dưới đây dùng JWT như core API, trừ preview public. Token lời mời chỉ gửi trong header `X-Invite-Token`, không đặt vào URL API hoặc query string.

| Endpoint | Quyền và đầu vào | Kết quả |
|---|---|---|
| POST /v1/plans/{id}/invite-links | Owner, verified email, Idempotency-Key; JSON max_uses (mặc định100,1..1000) |201 metadata và token; hết hạn7 ngày |
| GET /v1/plans/{id}/invite-links | Owner; cursor/limit | Metadata, không trả bearer token |
| DELETE /v1/invite-links/{id} | Owner | Thu hồi link |
| GET /v1/invite-preview | Public; X-Invite-Token | Chỉ title,activity,starts_at |
| POST /v1/invite-redeem | JWT; X-Invite-Token | plan_id; membership invited |
| GET /v1/plans/{id}/options | Member; cursor/limit | Snapshot options, vote_count,my_vote |
| POST /v1/plans/{id}/options | Owner; Idempotency-Key | JSON place_id hoặc name,address; tối đa20 options |
| PUT /v1/plans/{id}/vote | Member; JSON option_id | changed; một lựa chọn/người, có thể đổi |
| PUT /v1/plans/{id}/selection | Owner; If-Match version; JSON option_id | Plan và ETag mới, location từ snapshot |
| GET /v1/places | JWT; activity,kind,area,max_budget,people,cursor,limit | Catalogue page |

Preview thiếu/sai/hết hạn/hết lượt/thu hồi trả404 invite_unavailable. Thành viên hiện có có thể redeem lại link hết lượt mà không tiêu thêm lượt; link bị thu hồi/hết hạn vẫn bị từ chối. Người bị owner thu hồi membership không được dùng link để tự vào lại; owner có thể mời trực tiếp lại. Redeem và thu hồi khóa plan trước link để giữ thứ tự transaction nhất quán.

Preview public giới hạn120 request/phút theo direct peer IP trong process (không tin X-Forwarded-For); triển khai nhiều instance cần edge rate limiting theo client thực. Các endpoint authenticated vẫn dùng shared actor quota. Không log Authorization/X-Invite-Token hoặc response tạo link.

Voting chỉ mở khi plan inviting và chưa bắt đầu. Chốt/hủy/bắt đầu khóa thay đổi; edit location thủ công xóa selected_option_id. Mỗi option chụp tên/địa chỉ tại lúc thêm; đổi catalogue không thay location đã chọn. Ý tưởng hoạt động không có địa chỉ thực: owner cần bổ sung địa điểm cụ thể trước buổi hẹn.

Migration006 bổ sung snapshot `kind` (place/idea) vào option. Option custom là place; option từ catalogue lấy loại tại lúc thêm. Ideas vẫn nhận phiếu nhưng không được dùng làm location qua `/selection`: trả409 `venue_required`. Chốt mới cũng từ chối selected idea legacy; owner có thể nhập địa điểm cụ thể bằng PATCH (xóa selected_option_id) hoặc chọn một venue. Migration backfill kind từ catalogue hiện tại và không tự sửa lịch sử kèo đã confirmed; sửa tên/sức chứa của legacy confirmed vẫn được phép theo validation thông thường.

Catalogue kind=place có source_url HTTPS và checked_at; kind=idea là đề xuất nội dung. Budget null có nghĩa chưa xác minh, bị loại khi có filter max_budget. people là khoảng số người gợi ý, không phải thông tin chỗ trống thời gian thực. Không có endpoint admin catalogue public; dùng cmd/seed với operator role. INVITE_TOKEN_SECRET cần base64 của32 byte; chưa cấu hình trả503 khi tạo link, production từ chối khởi động nếu thiếu.
