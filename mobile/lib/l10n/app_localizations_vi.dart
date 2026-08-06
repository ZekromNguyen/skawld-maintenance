// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Vietnamese (`vi`).
class AppLocalizationsVi extends AppLocalizations {
  AppLocalizationsVi([String locale = 'vi']) : super(locale);

  @override
  String get appTitle => 'Skawld Maintenance';

  @override
  String get language => 'Ngôn ngữ';

  @override
  String get locale_system => 'Hệ thống';

  @override
  String get locale_english => 'English';

  @override
  String get locale_vietnamese => 'Tiếng Việt';

  @override
  String get login_subtitle =>
      'Hãy đăng nhập khi có kết nối. Công việc được phân công đã lưu sẽ vẫn sẵn sàng trong các lần mất kết nối sau.';

  @override
  String get login_button => 'Đăng nhập bằng OIDC';

  @override
  String get login_connecting => 'Đang kết nối…';

  @override
  String get list_assignedMaintenance => 'Công việc bảo trì được phân công';

  @override
  String get list_offlineBanner => 'SẴN SÀNG NGOẠI TUYẾN · CHỈ ĐỂ THAM KHẢO';

  @override
  String get list_syncNow => 'Đồng bộ ngay';

  @override
  String get list_empty =>
      'Không có công việc nào được lưu.\nKết nối một lần để tải công việc được phân công của bạn.';

  @override
  String get list_footer =>
      'Các thay đổi đang chờ được lưu trên thiết bị này và tự động đồng bộ.';

  @override
  String get signout_title => 'Đăng xuất?';

  @override
  String get signout_message =>
      'Phiên làm việc trên thiết bị này sẽ kết thúc. Công việc đã lưu vẫn ở trên thiết bị và đồng bộ sau khi bạn đăng nhập lại.';

  @override
  String get signout_cancel => 'Hủy';

  @override
  String get signout_confirm => 'Đăng xuất';

  @override
  String get signout_tooltip => 'Đăng xuất';

  @override
  String get detail_disclaimer =>
      'Skawld hướng dẫn và ghi lại công việc. PTW/LOTO bên ngoài vẫn giữ thẩm quyền.';

  @override
  String get detail_startOffline => 'Bắt đầu kiểm tra ngoại tuyến';

  @override
  String detail_requiresPrerequisite(String prerequisite) {
    return 'Yêu cầu $prerequisite';
  }

  @override
  String get detail_complete => 'Hoàn thành';

  @override
  String get measurement_record => 'Ghi số đo';

  @override
  String get measurement_vibrationVelocity => 'Vận tốc rung · mm/s';

  @override
  String get measurement_bearingTemperature => 'Nhiệt độ ổ trục · °C';

  @override
  String get measurement_exactValue => 'Giá trị chính xác';

  @override
  String get measurement_save => 'Lưu số đo trên thiết bị';

  @override
  String get copilot_title => 'Trợ lý có căn cứ bằng chứng';

  @override
  String get copilot_disclaimer =>
      'Chỉ dùng khi trực tuyến. Các đề xuất tham khảo không thể hoàn tất bước quy trình hoặc cấp phép PTW/LOTO.';

  @override
  String get copilot_busy => 'Đang truy xuất bằng chứng hợp lệ…';

  @override
  String get copilot_recommend => 'Đề xuất bước kiểm tra tiếp theo';

  @override
  String copilot_confidence(int percent) {
    return 'Độ tin cậy $percent%';
  }

  @override
  String get copilot_insufficient => 'THIẾU BẰNG CHỨNG';

  @override
  String copilot_unknowns(String value) {
    return 'Chưa rõ: $value';
  }

  @override
  String copilot_humanConfirmation(String provenance) {
    return '$provenance · cần xác nhận của con người';
  }

  @override
  String get observation_title => 'Nhận xét của kỹ thuật viên';

  @override
  String get observation_hint => 'Bạn đã quan sát thấy gì?';

  @override
  String get observation_save => 'Lưu ghi chú trên thiết bị';

  @override
  String get attachment_title => 'Bằng chứng ảnh hoặc ghi âm';

  @override
  String get attachment_subtitle =>
      'Tệp được xếp hàng riêng. Tải lên thất bại không xóa bản ghi kiểm tra.';

  @override
  String get attachment_choose => 'Chọn tệp từ thiết bị này';

  @override
  String get attachment_pickerLabel => 'Ảnh và ghi chú thoại';

  @override
  String get savedPendingSync => 'Đã lưu trên thiết bị · chờ đồng bộ';

  @override
  String get error_notAssigned =>
      'Chỉ công việc được phân công mới có thể bắt đầu';

  @override
  String get error_notInProgress =>
      'Công việc chưa trong trạng thái đang tiến hành';

  @override
  String error_prerequisiteUnverified(String prerequisite) {
    return '$prerequisite phải được cơ quan có thẩm quyền bên ngoài xác minh khi trực tuyến';
  }

  @override
  String get error_copilotOffline =>
      'Copilot yêu cầu kết nối trực tuyến đã xác thực. Hồ sơ hiện trường ngoại tuyến vẫn an toàn.';

  @override
  String get error_copilotEmpty => 'Copilot trả về phản hồi rỗng';

  @override
  String get error_attachmentSize =>
      'Tệp đính kèm phải có kích thước từ 1 byte đến 50 MiB';

  @override
  String get error_attachmentType => 'Loại tệp đính kèm không được hỗ trợ';

  @override
  String get config_invalid => 'Cấu hình máy tính để bàn không hợp lệ';

  @override
  String get config_hint =>
      'Kiểm tra skawld-config.json hoặc các biến môi trường được quản lý.';
}
