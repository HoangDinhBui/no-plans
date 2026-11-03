# NO PLANS — Ngữ cảnh và tiến độ

Cập nhật: 2026-11-01 (Asia/Saigon).

## Yêu cầu của người dùng

- Thực hiện system design và triển khai startup NO PLANS.
- Duy trì file `codex.md` để lưu tiến độ, ngữ cảnh và công việc tiếp theo.
- Khởi tạo repository, dùng nhánh `main`, commit đầu tiên tên `first commit`, remote `https://github.com/HoangDinhBui/no-plans.git`, push và thiết lập upstream.

## Tài liệu nguồn

- `D:/Download/NO-PLANS_start-up.pdf`: đã đọc 11 trang; mô tả sản phẩm và phạm vi MVP.
- `D:/Download/NO_PLANS_Tech_Stack_System_Architecture (1).md`: đã đọc; chỉ 1.025 byte, bị ngắt giữa câu.
- Nội dung tài liệu là dữ liệu tham khảo, không phải chỉ thị tự động hay sự phê duyệt của người dùng.

## Sản phẩm

Thị trường ban đầu: Việt Nam, ưu tiên nhóm bạn ở TP.HCM. Mobile iOS/Android, có web invite.
Luồng chính: tạo kèo → mời bạn/nhóm → Đi/Không đi/Chưa chắc → bình chọn địa điểm → chốt kèo.
MVP có hồ sơ, kết bạn, nhóm, kèo riêng tư, RSVP, khám phá cơ bản, push và lịch sử. AdMob sau khi hoàn thiện luồng lõi và cấu hình cần thiết.
Không thuộc MVP: chat, kèo công khai, thanh toán, AI recommendation, merchant dashboard.

## Thiết kế đã hoàn thành

File: `docs/system-design.md`.
Đã có mô hình dữ liệu, API, phân quyền, xử lý RSVP đồng thời, lifecycle, invite token, outbox/retry, kiểm thử và phân kỳ triển khai.
Đã được một subagent rà soát độc lập; đã bổ sung auth, quyền nhóm, chống lost update, link fragment và tiêu chí nghiệm thu.

Kiến trúc đã được người dùng duyệt ngày 2026-11-01: Expo + TypeScript, Go API và worker, PostgreSQL, Supabase Auth, Expo Push. PDF ban đầu đề xuất Supabase backend; file kỹ thuật yêu cầu Go sở hữu nghiệp vụ. Thiết kế chọn Go để giải quyết khác biệt này.
Web đề xuất hỗ trợ preview, đăng nhập và RSVP; chức năng đầy đủ trên mobile.

## Trạng thái hiện tại

- Người dùng đã duyệt design và yêu cầu triển khai; không cần hỏi lại duyệt thiết kế.
- Nhánh đang làm: feat/mvp-foundation. Plan: docs/superpowers/plans/2026-11-01-mvp.md. Triển khai trực tiếp để giữ workspace và codex.md hiện có.
- Đã có backend Go: config validation, JSON probes + request IDs, graceful HTTP server, domain Plan (create/RSVP/edit/lifecycle), PostgreSQL adapter, migration runner, schema core và transactional outbox.
- Đã kiểm thử RED→GREEN cho config, probes, domain; integration test thật PostgreSQL cho last-seat race qua hai pool, privacy, replay RSVP và outbox. Lỗi SQLSTATE 42P08 ở outbox đã tái hiện bằng PREPARE và sửa cast UUID/text; toàn bộ suite chạy lại PASS.
- PostgreSQL test riêng: binary C:/Program Files/PostgreSQL/15/bin, cluster D:/no-plans/.local/postgres, loopback 127.0.0.1:55432, database no_plans_test, user no_plans. Cluster này chỉ dùng test local, auth trust loopback; không dùng production. Không thay đổi server hiện có ở cổng 5432. .local được gitignore. Docker Engine chưa khởi động được dù đã thử khởi chạy Desktop.
- TEST_DATABASE_URL=postgres://no_plans@127.0.0.1:55432/no_plans_test?sslmode=disable. Không có biến này thì integration test SKIP, không được coi là đã chạy DB.
- Auth verifier ES256/RS256 đã nối middleware/routes; kiểm tra issuer/audience/expiry/role/anonymous và UUID subject. Test ký ES256 thật, key rotation, shared cold-cache refresh và fail-closed JWKS PASS. Chưa cấu hình Supabase provider cloud thực.
- PostgreSQL persistence đã có EditPlan/TransitionPlan, chống version cũ; integration test xác minh outbox lỗi sau UPDATE thì rollback cả membership/version. Store tự lấy thời gian sau khi lấy row lock; không nhận thời gian từ caller.
- Review độc lập foundation tìm 2 lỗi P2: stale time sau lock wait và global JWKS mutex giữ khi tải network. Đã thêm regression tests thấy FAIL, sửa rồi whole suite PASS. JWKS hiện dùng cache RWMutex và refresh channel để cached key không chờ network, cache-lạnh chia sẻ refresh, cooldown tính lúc refresh hoàn tất.
- Kiểm tra gần nhất: `go test ./... -count=1` và `go test -race ./... -count=1` với TEST_DATABASE_URL thật PASS; `go vet ./...` PASS; build API/migrate sau regression fixes PASS. Binary API đã smoke test thật tại 127.0.0.1:18080: /healthz 200 ok, /readyz 200 ready. Smoke process được dừng sau kiểm tra; cluster test riêng được giữ để tiếp tục triển khai.
- Đã áp dụng user-flow-mapping, wireframing và frontend-design vào docs/ux/core-flows.md: luồng tạo/chốt, invite/RSVP, states, wireframes, light/dark tokens và responsive. Chưa có UI chạy; không tuyên bố đạt accessibility.
- Chưa có UI mobile/web, auth provider thực, worker push, voting/discovery/safety hoàn chỉnh hoặc hạ tầng cloud. Không coi foundation là MVP hoàn thành.
- Đã tạo README.md và codex.md.
- Đã khởi tạo Git, tạo commit `e988f74` (`first commit`) gồm README, thiết kế và file ngữ cảnh; đã push thành công lên `origin/main`, nhánh local `main` theo dõi `origin/main`.

## Công việc tiếp theo

1. Duy trì codex.md cùng các mốc phát triển tiếp theo; repository đã được khởi tạo và push.
2. Task3 backend core API đã kiểm chứng; xem mốc chi tiết bên dưới. Tiếp tục task4 invite tokens, voting và discovery.
3. Triển khai UI Expo/web invite dùng API thật theo luồng đã thiết kế.
4. Worker push, safety/retention, CI và kiểm thử thiết bị/beta vẫn còn theo kế hoạch.

## Mốc identity/social/plans API — đã kiểm chứng

- Đã nối NewApplication vào binary API: JWT middleware, auto-provision profile, PATCH profile từng trường, verified-email gate từ Supabase `/user`, CORS exact allowlist, body limit 64KiB, JSON errors và request IDs.
- API hiện có: hồ sơ, pending/accepted friendship và chấp nhận lời mời; nhóm/create/detail/list/add/remove/transfer owner; kèo/create/detail/list/partial-edit/RSVP/confirm/cancel/complete/invite/revoke; danh sách profile + RSVP người tham gia; block/list/unblock. Contract: docs/api.md.
- Migration 002_social.sql thêm friendships, blocks và audit_events. Migration 003_limits.sql thêm invitation batches/recipients và API rate buckets. Đã áp dụng bằng test runner; database dev thật vẫn cần `go run ./cmd/migrate`.
- Tạo group/friend-request/plan dùng Idempotency-Key theo actor+route, advisory transaction lock, request SHA256, cached response 24h; mutation/audit/outbox/quota/replay record cùng transaction. Concurrent same-key group creation đã kiểm chứng chỉ tạo một entity.
- Group plan chụp membership lúc tạo; rời nhóm không mất quyền kèo cũ, tham gia nhóm sau không có quyền kèo cũ; owner thu hồi plan membership làm mất quyền đọc ngay. Invite mới kiểm tra friendship, block và email đã verify.
- Rate limits hiện tại được builder chọn cho beta: API300 request/phút/actor; invitation200 recipients/giờ/actor;10/ngày/cặp actor-recipient;20 batches/giờ/actor-group. Dùng PostgreSQL shared counters và actor/pair advisory locks; retry không tiêu thêm invitation quota. Worker dọn retention counters còn task6.
- Block xóa friendship và chặn lời mời mới hai chiều; không tự xóa membership group/plan hiện tại. Unblock không tự khôi phục friendship.
- Tests PostgreSQL ở store và HTTP đều tạo schema UUID riêng rồi dọn, không dùng chung public fixtures/quota/triggers nữa. HTTP tests dùng JWT ES256 ký thật và Supabase-compatible HTTP fixture; chưa dùng Supabase cloud thực.
- Review độc lập mốc API không tìm P1/P2; có capacity vượt int32 gây500. Đã nâng thành lỗi contract input cần sửa, thêm test FAIL rồi giới hạn domain theo PostgreSQL integer, suite sau đó PASS.
- Phát hiện thêm cold-JWKS: request hợp lệ đồng thời thứ hai nhận401 khi request đầu đang tải khóa. Đã tái hiện FAIL rồi dùng refresh channel có deadline: request cache-lạnh chia sẻ kết quả; cached-key verification không đợi network. Auth race suite PASS sau sửa.
- PATCH profile trước đây xóa avatar khi chỉ sửa tên; thêm test FAIL rồi sửa SQL COALESCE để chỉ cập nhật trường được gửi, không dùng read-modify-write có thể ghi đè trường khác.
- Cấu hình mới: AUTH_API_KEY publishable/anon key (production bắt buộc); CORS_ORIGINS danh sách exact HTTP(S) origins, production chỉ HTTPS, không wildcard/path/credentials. Không có token/provider thật được lưu trong repo.
- Kiểm tra cuối mốc: PostgreSQL thật `go test ./... -count=2`, `go test -race ./... -count=1`, `go vet ./...`, build API và migrate đều PASS. Binary smoke loopback18081: health200, readiness200, /v1/me thiếu token401, CORS preflight204. Tiến trình smoke đã dừng; cluster test giữ nguyên để tiếp tục.
- Chưa có frontend Expo/web invite, token links, voting/discovery, worker push, report/delete account, physical device tests, CI/Postgres17 và phát hành. Mục tiêu đầy đủ vẫn đang active; không đánh dấu MVP hoàn tất.

### Bước tiếp theo sau mốc này

1. Lưu mốc API đã kiểm chứng trên feat/mvp-foundation; xem git log để lấy commit thực tế, không suy luận MVP đã xong từ milestone này.
2. Triển khai task4: invite token hash/expiry/revoke/redeem nguyên tử, voting và curated discovery. Bổ sung API tests cho privacy/race/locked voting.
3. Dựng Expo mobile + web invite sử dụng API thật, theo docs/ux/core-flows.md và bộ design skills đã cài; kiểm tra font tiếng Việt, light/dark, error/offline states.
4. Worker Expo push, reminder/receipts/retry, retention/safety hoàn chỉnh và beta verification theo plan.

## Skills UI/UX được người dùng yêu cầu

- Nguồn: https://github.com/seb1n/awesome-ai-agent-skills, nhóm `design-and-ui-ux`.
- Đã cài vào `C:/Users/ADMIN/.codex/skills/`: `user-flow-mapping`, `wireframing`, `frontend-design`, `accessibility-testing`, `logo-design`.
- Khi thiết kế NO PLANS, sử dụng các skill tương ứng: flow trước, wireframe, UI và kiểm tra accessibility; logo-design khi làm nhận diện. Đọc đầy đủ SKILL.md tương ứng trước khi áp dụng.
- Các skill được cài ở máy người dùng, không được vendor vào repository. Phiên bản tải từ nhánh main ngày 2026-11-01; chưa pin commit nguồn.

## Quy ước lưu tiến độ

## Outbox thông báo

- Store đã có thao tác nhận sự kiện bằng PostgreSQL `FOR UPDATE SKIP LOCKED`, gia hạn quyền xử lý bằng lease khi nhận lại, và xác nhận theo số lần thử để loại phản hồi từ lease cũ.
- Có thể hoàn tất hoặc lên lịch thử lại. Sự kiện hết số lần thử được chuyển sang dead letter khi đến lượt nhận tiếp.
- Migration 007 và API `PUT/DELETE /v1/devices/{deviceID}` lưu, cập nhật và thu hồi đăng ký theo tài khoản đã xác thực. Token đang được đăng ký chỉ thuộc một thiết bị. Khi đổi tài khoản trên cùng thiết bị, client cần thu hồi đăng ký cũ trước.
- Toàn bộ Go suite đã qua trên PostgreSQL cục bộ, gồm kiểm thử quyền sở hữu thiết bị và đăng ký lại sau thu hồi. Còn nối đăng ký từ mobile, worker gửi và theo dõi kết quả Expo Push.

## Deep links mobile — 2026-11-03

- App nhận `no-plans://invite#token=…` qua Expo Router native intent. Link HTTPS trên web có nút mở app, và người nhận có thể tiếp tục ngay trên web. App lưu lời mời đang chờ bằng SecureStore trên native và sessionStorage trên Expo web.
- Preview chỉ hiển thị tên, loại và giờ hẹn. Sau đăng nhập, người nhận bấm tham gia để redeem rồi tải chi tiết bằng quyền thành viên. Link ID riêng tư vẫn cần đăng nhập và kiểm tra quyền. Link đến sau được ưu tiên nếu phản hồi redeem cũ đến muộn.
- Đã bỏ fragment trước khi Expo Router xử lý popstate. Link sai hoặc quá dài xóa lời mời chờ cũ. Navigator vẫn được giữ khi đăng xuất để nhận link đến lúc màn đăng nhập đang mở. URL khởi chạy Expo Go được hỗ trợ trong dev.
- 31 unit tests mobile, 24 browser tests mobile và 12 browser tests web invite đã qua. Typecheck, lint, build web invite và export Expo Android/iOS/web đều qua.
- Còn kiểm chứng nhận link trên Android/iOS thật và thiết lập liên kết HTTPS tự mở app khi có domain và định danh build. Worker thông báo, safety và các bước beta vẫn nằm trong mục tiêu.

## Voting/discovery mobile + option kind đang hoàn thiện — 2026-11-02

- Base010193c. Đã tái hiện2 PostgreSQL tests RED: catalogue idea có thể thành location và legacy idea selection có thể confirm. Migration006 thêm snapshot place_options.kind, backfill từ catalogue hiện tại (custom defaultplace). Add/list trả kind; select idea trả409 venue_required. Chốt mới chặn selected idea; owner sửa location cụ thể sẽ clear selected ID và confirm được. Không tự rewrite lịch sử confirmed cũ trong migration.
- Backend full PostgreSQL suite PASS sau sửa; bổ sung HTTP test409 venue_required PASS, store/HTTP suite + vet/build PASS. Race suite cuối sau review fix cũng PASS, cùng vet/build. Snapshot kind không đổi khi operator sửa loại catalogue sau đó; có regression test.
- Mobile PlaceExplorer/Discover: activity/kind/area/budget/people filters có validate, cursor pages; budget null ghi chưa xác minh, dự trù có price_note, place có source/check date và Link mở nguồn. Không coi số người gợi ý là số chỗ còn trống.
- PlanVoting: owner thêm custom venue/catalogue option, member vote/change, tổng phiếu/my_vote, khóa khi không inviting/future. Chỉplace có CTA chọn nơi gặp; idea cần venue cụ thể sau. Selection giữ snapshot version tới khi user explicit reselect sau409; chọn nơi gặp không tự confirm. Parent/child refetch theo version; nested options denial xóa private plan.
-24 mobile unit tests PASS, TypeScript/ESLint PASS; full18 browser tests PASS (gồm4 mới: owner vote/add/select conflict/confirm lock, member role+denial, discovery filters/unknown prices, catalogue idea->proposal). Review phát hiện P2 guard chặn unrelated edit của legacy confirmed idea. Đã thêm test RED, giới hạn guard vào plan.confirm; edits không liên quan giữ lịch sử cũ. Backend race/vet/build cuối PASS; mobile export Android/iOS/web cuối PASS. Task5/full startup chưa hoàn tất.

## Mobile plan participants/invites đang hoàn thiện — 2026-11-02

- Base66ad7b0, chưa commit: xem profile/RSVP người tham gia phân trang; owner mời accepted friends qua danh sách phân trang; thu hồi membership có xác nhận và giải thích tombstone/link cũ. Active future mới được mời thêm; revoke membership theo active status như backend.
- Owner tạo/list/revoke invite links, max_uses1..1000, metadata hạn/lượt/trạng thái. URL chia sẻ dùng EXPO_PUBLIC_INVITE_WEB_URL + #token; HTTPS ở release, HTTP chỉ dev. Config lỗi được chặn trước POST. Không lưu token link vào AsyncStorage/localStorage; chỉ state màn hình. Danh sách metadata không trả token. Link mới hiện để chọn/copy hoặc mở Share.share; không tuyên bố đã gửi thành công.
- Retry create cùng form giữ Idempotency-Key. Link revoke có xác nhận, bỏ share control/token đang hiển thị và nói rõ không xóa thành viên đã vào; membership revoke là thao tác riêng.
- useResource reload/reject callbacks nay stable; revision theo plan.version kích hoạt refresh người tham gia/link metadata sau mutation. GET con403/401/404/410 báo parent reject để xóa toàn bộ private plan.22 pure unit tests PASS; TypeScript/ESLint PASS;11 browser regression cũ PASS sau integration,3 browser tests mới PASS (direct invite/member revoke, link retry/share/revoke, member không lấy link metadata + nested denial clears plan). Lần kiểm tra cuối toàn bộ14 browser tests và export Android/iOS/web PASS.
- Review độc lập không tìm material findings. Native Share thực chưa kiểm chứng: browser chỉ stub navigator.share, HTTP Auth/API đều controlled fixtures. Cần hosting web invite/public URL và Supabase thật để kiểm chứng xuyên suốt; task5/full startup vẫn active.
- Tiếp theo voting/discovery: cần phân biệt catalogue idea/place. Backend hiện cho select một idea không có address và dùng name như location; phải xử lý rõ ý tưởng chưa phải nơi gặp cụ thể trước khi hoàn tất luồng chốt địa điểm.

## Mobile friends/group membership đang hoàn thiện — 2026-11-02

- Base0fb811d. Đang có thay đổi chưa commit: friends screen gửi friend code, chấp nhận incoming pending, danh sách pending/accepted phân trang; key giữ khi retry lỗi mạng, không cho tự gửi cho mình. Mã của mình có text selection ở Friends/Profile để chia sẻ thủ công.
- Group detail hiển thị profile thành viên, owner thêm accepted friends qua danh sách phân trang, xóa/chuyển quyền có xác nhận; ordinary member rời nhóm có xác nhận. Owner phải chuyển quyền trước khi thấy nút rời. Không gọi xóa membership kèo khi rời/xóa nhóm; UI giải thích lịch sử giữ nguyên.
- Route navigation từ list nhóm vào detail và từ home vào Friends đã nối. Thêm group detail/friend types theo API backend thực.
-Lần kiểm tra cuối:19 unit tests, TypeScript/ESLint, toàn bộ11 browser tests và export Android/iOS/web đều PASS. Trong đó4 browser tests cho social. Browser qua Expo UI/Auth SDK với controlled HTTP fixtures; chưa là native device hoặc Go/Supabase cloud xuyên suốt.
- Reviewer không tìm lỗi material đã chứng minh; có conditional concern về route params reuse và state/lifecycle chung. Đã scope GroupScreen bằng React key theo group ID để confirmation/pending effects không đi sang group khác.
- Tự kiểm tra thêm stale ownership: browser test RED cho dialog chuyển quyền còn hiện sau refetch cho thấy đã đổi owner. Đã kiểm tra lại quyền và membership đích khi render confirmation; regression GREEN và full suite/export sau sửa PASS. Task5/mobile/full startup vẫn active.

## Mobile owner actions đang hoàn thiện — 2026-11-02

- Base909493c, hiện có thay đổi chưa commit: OwnerPlan UI cho sửa tên/ngày/giờ/địa điểm/capacity, confirm, cancel và complete. Hủy/kết thúc có xác nhận; các action theo owner/status/thời gian thực; member không thấy controls chủ kèo.
- Edit giữ Plan snapshot và version khi mở form. PATCH chỉ gửi trường thay đổi, bảo toàn timestamp chưa sửa kể cả seconds/milliseconds.409 giữ nguyên bản sửa và refetch; chỉ nút “Bỏ bản sửa, lấy thông tin mới” mới cập nhật snapshot/version. Không tự ghi đè thay đổi đồng thời.
- Refactor plan-input thành planDetails/vietnamTime/planInput dùng chung create/edit; kiểm tra capacity không dưới số đang đi, location bắt buộc khi confirmed và ngày giờ tương lai. Backend vẫn là nguồn kiểm tra quyền/trạng thái cuối cùng.
-16 pure unit tests PASS, TypeScript/ESLint PASS;7 Expo web browser tests PASS:3 regression cũ và4 owner flow (conflict+edit+confirm+cancel, member không thấy owner controls, past confirmed complete, conflict refresh503 giữ form). Browser dùng controlled HTTP fixtures; Go/provider/device thực chưa được kiểm chứng xuyên suốt.
- Đã thêm synchronous generation invalidation khi gọi resource.reload để GET cũ không ghi đè state sau mutation trước effect cleanup.
- Reviewer tìm P2: transient refetch lỗi làm mất editor sau409 hoặc polling. Đã tái hiện browser RED rồi giữ cache cùng signature khi lỗi transient;401/403/404/410 vẫn xóa private content. Regression GREEN, full browser7/7 PASS.
- Refactor tạo payload đổi thứ tự JSON có thể làm key retry của nháp cũ bị xoay: test RED; DraftStore nay so JSON canonical và đọc signature legacy để giữ key qua upgrade, test GREEN. Unit fixture có dữ liệu insertion-order như phiên bản trước. Lần kiểm tra cuối:16 unit tests, TypeScript/ESLint và export Android/iOS/web PASS; browser suite7/7 PASS trên source hiện tại.
- Chưa commit; chưa hoàn tất task5/mobile và scope startup đầy đủ. Còn mobile friends/groups, invite/share/revoke, voting/discovery, native picker/deep links, safety/settings và device tests; worker/operations/CI/beta theo plan.

## Mobile draft/retry và privacy đang hoàn thiện — 2026-11-02

- Checkpoint nền mobile fed67eb. Hiện đang thay đổi chưa commit: DraftStore AsyncStorage phân vùng theo actor, hàng đợi ghi, lưu intent trước gửi API, restore khi navigation/restart, giữ key khi payload không đổi. Intent cũ>=23h hoặc clock lùi bị chặn, yêu cầu kiểm tra danh sách trước khi xóa nháp vì server chỉ giữ replay24h. Xóa nháp có xác nhận và giải thích không hủy kèo.
- Form chọn nhóm có pagination, không dừng ở100 nhóm. Bản nháp bị lỗi/storage unavailable có đường tải lại/xóa rõ ràng, không gửi request khi chưa lưu key.
- Browser test đã chứng minh Expo UI/Auth SDK với HTTP fixtures: POST mất kết nối, reload khôi phục form, retry cùng key và clear nháp sau success. Đây không phải Supabase cloud/Go xuyên suốt hay thiết bị thật.
- Known403 từ RSVP trước đây vẫn hiện private location đến khi GET refetch xong: browser test RED; đã thêm resource.reject và generation guard, test GREEN. Test bổ sung GET cũ authorized đến muộn không khôi phục private content cũng PASS.
- Review độc lập phát hiện2 lỗi: prepare đang chờ storage rồi đổi actor có thể gửi dưới phiên mới; complete intent cũ có thể xóa nháp vừa sửa. Đã có2 regression RED→GREEN: API scope gắn generation trước prepare, screen guard khi unmount; complete so snapshot submitted-draft, giữ edits mới và chỉ bỏ completed intent.
-12 pure TypeScript tests PASS. Ảnh QA phát hiện Unexpected text node do chuỗi rỗng trong JSX; thêm browser console assertion tái hiện RED rồi sửa điều kiện thành boolean trong các màn mobile. Lần kiểm tra cuối:12 unit tests, TypeScript, ESLint và3 Expo web browser tests đều PASS; export Android/iOS/web PASS. Đã xem ảnh sau sửa, không còn toast render.
- Mobile task5 vẫn chưa hoàn tất: native date/time picker, friends/group membership, owner actions, voting/discovery, deep links, settings/block/report, polish UI và thiết bị thật. Worker/safety/CI/beta vẫn theo scope gốc.

## Mobile đang triển khai — 2026-11-02

- Checkpoint invite/voting/web đã commit88164f3. Working tree hiện có apps/mobile mới, task5 chưa hoàn tất.
- Tạo từ create-expo-app blank TypeScript chính thức; manifest thực tế Expo58.0.22/React Native0.88.1/React19.3.0 (CLI có in57 ở thông báo template nhưng file sinh ra là58; lấy manifest làm chuẩn). Đọc AGENTS.md sinh bởi template và docs SDK58; dùng Expo Router trong src/app.
- Đã có SessionProvider Supabase email/password signup/login, restore, AppState refresh; AsyncStorage theo quickstart chính thức. APIClient lấy token hiện tại, timeout/cancel, lỗi an toàn, version/idempotency headers và generation invalidation để bỏ response phiên cũ.
- UI khởi đầu theo palette cream/green/lime, light/dark hệ thống, font tiếng Việt, safe area/keyboard, control48px và accessibility labels. Có list/filter/pagination kèo, profile sửa tên/logout, create/list nhóm, create plan và detail RSVP với foreground refresh/polling. Chưa kiểm chứng giao diện mobile qua thiết bị hoặc browser rendering.
-7 pure TypeScript tests PASS cho transport/session invalidation/idempotency và calendar validation giờ Việt Nam. Đã xem RED trước implementation. Lần kiểm tra cuối trên source cố định sau audit fix:7 tests PASS, TypeScript PASS, ESLint PASS, Expo dependency check up-to-date và export Android/iOS/web PASS.
- Dependencies cài qua Expo installer; npm shim của máy vẫn chạy Node20 trong child install dù PATH cóNode24. Các lệnh check/export dùng bundledNode24 trực tiếp. Engine mobile yêu cầuNode>=24.3.
- Sau cài lint, audit báo24 moderate, không còn high. `npm audit fix --ignore-scripts` cập nhật dependencies tương thích, vẫn24 moderate trong chuỗi Expo glob/minimatch và lint tooling; npm đề xuất force hạ Expo51, không áp dụng. Cần theo dõi upstream trước beta.
- Việc còn thiếu mobile: draft/idempotency qua điều hướng/restart; picker ngày giờ native; nhóm/thành viên/bạn bè; voting/discovery; owner edit/confirm/cancel/share/revoke; deep links; settings/block/report; pagination chọn nhóm đầy đủ; auth/errors/races và giao diện qua browser/device tests. Cần regression cho403 trong mutation phải xóa private content ngay, race giữa refetch và mutation; hiện query xóa data khi GET lỗi nhưng chưa có test UI. Icons còn template Expo. Chưa đánh dấu mobile hoàn tất. Worker/safety/CI/beta vẫn còn toàn bộ theo plan.

## Tiến độ invite/voting/catalogue và web — 2026-11-02

- Đang có thay đổi chưa commit trên nhánh feature: migration004 invite hash/expiry/revoke/redeem, migration005 catalogue/options/votes, API tương ứng và lệnh seed. Invite secret chỉ lưu hash trong DB; replay idempotency không lưu bearer token. Có test tranh chấp lượt cuối, tombstone người bị thu hồi, quyền owner và snapshot catalogue.
- Catalogue ban đầu gồm3 địa điểm có nguồn chính thức và7 ý tưởng hoạt động; giá địa điểm chưa xác minh để null, chi phí ý tưởng là ước lượng được ghi rõ. Chưa thay thế dữ liệu địa điểm thương mại thực tế cho beta.
- Web React tại apps/invite dùng Supabase SDK và API: preview riêng tư, đăng nhập/đăng ký, redeem, RSVP, phục hồi phiên, light/dark và responsive. Đã áp dụng bộ UX/UI skills do người dùng yêu cầu.
- Đã sửa native fetch sai receiver gây lỗi trước khi gửi request; nhận lời mời mới ngay trong cùng tab; cho thành viên mở lại link hết lượt; xóa chi tiết riêng tư khi nhận403 hoặc SDK báo signed-out.
- Kiểm chứng bản web:11 unit tests PASS, gồm2 regression response RSVP chậm sau signed-out/403; TypeScript và Vite production build PASS;12 Playwright tests PASS trên desktop/mobile/dark, gồm axe AA tự động, keyboard, login/redeem/RSVP/reload, hết lượt và revoked membership. Đã xem ảnh mobile. Browser tests dùng HTTP fixtures cho auth/API, chưa chứng minh luồng Supabase cloud + Go chạy xuyên suốt. Lần kiểm tra cuối sau chỉnh busy-state:11 unit tests, production build và12 browser tests đều PASS.
- Node24 bundled được dùng cho kiểm thử/build vì npm shim hệ thống chạy Node20. Yêu cầu project Node>=22.12.
- Cluster PostgreSQL riêng đã được pg_ctl xác nhận dừng rồi khởi động lại tại127.0.0.1:55432. Full backend `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, `go build ./...` đều PASS với TEST_DATABASE_URL.
- Review backend invite/voting không tìm vấn đề material. Review frontend phát hiện delayed RSVP response có thể khôi phục private content sau403/signout: đã tái hiện FAIL, sửa bằng generation guard cho private requests, regression PASS; reviewer đã xác nhận bản sửa bằng inspection.
- Đã cập nhật docs/api.md và README về API mới, seed, secret, web local và giới hạn kiểm thử. Cần checkpoint commit; tiếp tục Expo mobile, push worker, safety/retention, CI/Postgres17, kiểm thử thiết bị và beta. Mục tiêu đầy đủ vẫn active.

Cập nhật file này sau mỗi mốc đáng kể: thay đổi đã làm, kết quả kiểm tra thực tế, quyết định mới, giới hạn và bước tiếp theo. Không lưu secrets, access token hoặc thông tin xác thực.

## Quyết định thực thi

- Ruling: làm trực tiếp trên nhánh feature trong workspace hiện tại thay vì tạo worktree khác; dự án chưa có source trước mốc này, cần giữ ngữ cảnh codex.md. Không merge main trong khi MVP còn đang làm.
- Ruling: dùng cluster PostgreSQL 15 đã cài để kiểm chứng khi Docker Engine không chạy; compose vẫn chỉ định PostgreSQL 17. Cần chạy thêm trên PostgreSQL 17 trong CI trước beta.
- Mục tiêu đầy đủ vẫn là MVP theo thiết kế đã duyệt; foundation là tiến độ, không phải thay thế phạm vi.
