class YTReq {
  String cmd;
  String data;
  YTReq(this.cmd, this.data);
  Map<String, dynamic> toJson() => {
        'cmd': cmd,
        'data': data,
  };
}

class YTRep {
  String cmd;
  String data;
  YTRep(this.cmd, this.data);
  factory YTRep.fromJson(dynamic json) {
    return YTRep(json['cmd'] as String,json['data'] as String);
  }

}