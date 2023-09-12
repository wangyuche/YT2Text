import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:yt2text/src/main_page/main_page.dart';

void main() {
  debugPaintSizeEnabled = false;
  runApp(MyApp());
}

class MyApp extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      builder: (context, widget) {
        return MediaQuery(
          data: MediaQuery.of(context).copyWith(textScaleFactor: 1.0),
          child: widget!,
        );
      },
      debugShowCheckedModeBanner: false,
      home: MainPageScreen(),
      routes: <String, WidgetBuilder>{},
    );
  }
}
