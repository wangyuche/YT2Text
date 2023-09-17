import 'package:flutter/material.dart';
import 'package:web_socket_channel/web_socket_channel.dart';
import 'package:yt2text/src/struct/struct.dart';
import 'dart:convert';
import 'dart:html' as html;

class MainPageScreen extends StatefulWidget {
  @override
  _MainPageState createState() => _MainPageState();
}

class _MainPageState extends State<MainPageScreen> {
  late List<String> _timetext;
  late List<String> _notimetext;
  late bool istimestamp = true;
  late WebSocketChannel channel;
  late TextEditingController YTURL = TextEditingController();
  late String _process;
  late ScrollController _scrollController = ScrollController();
  late String filetext;

  @override
  void initState() {
    filetext = "";
    _process = "閒置中";
    _timetext = [];
    _notimetext = [];
    //YTURL.text = "https://www.youtube.com/watch?v=BaW_jenozKc";
    //YTURL.text = "https://www.youtube.com/watch?v=C3-rChnzCw4";
    channel = WebSocketChannel.connect(
        Uri.parse('ws://' + Uri.base.host + ':8080/ws'));

    channel.stream.listen(
      (message) {
        print(message);
        YTRep rep = YTRep.fromJson(jsonDecode(message));
        switch (rep.cmd) {
          case "getcaptions":
            setState(() {
              _process = "轉出字幕中";
              _timetext.add(rep.data);
              List<String> result = rep.data.split('] ');
              if (result.length >= 2) {
                _notimetext.add(result[1]);
              }
              _scrollController.animateTo(
                  _scrollController.position.maxScrollExtent,
                  duration: const Duration(milliseconds: 500),
                  curve: Curves.fastOutSlowIn);
            });
            break;
          case "idle":
            setState(() {
              _process = "閒置中";
            });
            break;
          case "downloadyt":
            setState(() {
              _process = "下載影片中";
            });
            break;
        }
      },
      onDone: () {
        print('ws channel closed');
      },
      onError: (error) {
        print('ws error $error');
      },
    );
    super.initState();
  }

  @override
  void dispose() {
    channel.sink.close();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
        theme: ThemeData(
          primarySwatch: Colors.grey,
          primaryColor: Colors.white,
        ),
        debugShowCheckedModeBanner: false,
        builder: (context, widget) {
          return Scaffold(
              backgroundColor: Colors.white,
              resizeToAvoidBottomInset: false,
              body: Container(
                  margin: EdgeInsets.all(20),
                  width: MediaQuery.of(context).size.width,
                  height: MediaQuery.of(context).size.height,
                  child: Column(
                    mainAxisAlignment: MainAxisAlignment.start,
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: <Widget>[
                      Row(
                        children: <Widget>[
                          Expanded(
                            child: Container(
                              //alignment: Alignment.center,
                              padding: EdgeInsets.all(10),
                              width: MediaQuery.of(context).size.width * 0.7,
                              height: MediaQuery.of(context).size.height * 0.1,
                              child: TextField(
                                controller: YTURL,
                                textAlign: TextAlign.center,
                                style: TextStyle(
                                    fontSize: 40 *
                                        (MediaQuery.of(context).size.height /
                                            980)),
                                decoration: InputDecoration(
                                  border: OutlineInputBorder(),
                                  hintText: 'Enter Youtube URL',
                                  //contentPadding: EdgeInsets.symmetric(vertical: MediaQuery.of(context).size.height * 0.02),
                                ),
                              ),
                            ),
                          ),
                          Container(
                            padding: EdgeInsets.all(10),
                            width: MediaQuery.of(context).size.width * 0.2,
                            height: MediaQuery.of(context).size.height * 0.1,
                            child: ElevatedButton(
                              child: Text(
                                "開始",
                                style: TextStyle(
                                    fontSize: 20 *
                                        (MediaQuery.of(context).size.height /
                                            980)),
                              ),
                              style: ElevatedButton.styleFrom(
                                elevation: 0,
                              ),
                              onPressed: () {
                                setState(() {
                                  _timetext = [];
                                  _notimetext = [];
                                });
                                YTReq ytreq = YTReq("downloadyt", YTURL.text);
                                channel.sink.add(jsonEncode(ytreq.toJson()));
                              },
                            ),
                          ),
                        ],
                      ),
                      Row(
                        mainAxisAlignment: MainAxisAlignment.center,
                        children: <Widget>[
                          Text(
                            '顯示時間',
                            style: TextStyle(
                                fontSize: 20 *
                                    (MediaQuery.of(context).size.height / 980)),
                          ),
                          Switch(
                            value: istimestamp,
                            activeColor: Colors.white,
                            activeTrackColor: Colors.blue,
                            inactiveThumbColor: Colors.white,
                            inactiveTrackColor: Colors.grey,
                            onChanged: (bool value) {
                              setState(() {
                                istimestamp = value;
                              });
                            },
                          ),
                          SizedBox(
                            width: MediaQuery.of(context).size.width * 0.1,
                          ),
                          Text(
                            _process,
                            style: TextStyle(
                                fontSize: 20 *
                                    (MediaQuery.of(context).size.height / 980)),
                          ),
                          SizedBox(
                            width: MediaQuery.of(context).size.width * 0.1,
                          ),
                          IconButton(
                              icon: Icon(Icons.download),
                              onPressed: () {
                                filetext="";
                                var _t = _timetext;
                                if (!istimestamp) {
                                  _t = _notimetext;
                                }
                                for (var t in _t) {
                                  filetext = filetext + t + "\r\n";
                                }
                                final bytes = utf8.encode(filetext);
                                final blob = html.Blob([bytes]);
                                final url =
                                    html.Url.createObjectUrlFromBlob(blob);
                                final anchor = html.document.createElement('a')
                                    as html.AnchorElement
                                  ..href = url
                                  ..style.display = 'none'
                                  ..download = 'some_name.txt';
                                html.document.body?.children.add(anchor);
                                anchor.click();
                                html.document.body?.children.remove(anchor);
                                html.Url.revokeObjectUrl(url);
                              })
                        ],
                      ),
                      Container(
                        width: MediaQuery.of(context).size.width,
                        height: MediaQuery.of(context).size.height * 0.5,
                        margin: EdgeInsets.all(10),
                        decoration: BoxDecoration(
                          color: Colors.grey,
                        ),
                        child: ListView.builder(
                          controller: _scrollController,
                          itemCount: istimestamp == true
                              ? _timetext.length
                              : _notimetext.length,
                          itemExtent: MediaQuery.of(context).size.height * 0.05,
                          itemBuilder: (BuildContext context, int index) {
                            return Text(
                              istimestamp == true
                                  ? _timetext[index]
                                  : _notimetext[index],
                              style: TextStyle(
                                  fontSize: 30 *
                                      (MediaQuery.of(context).size.height /
                                          980)),
                            );
                          },
                        ),
                      ),
                    ],
                  )));
        });
  }
}
