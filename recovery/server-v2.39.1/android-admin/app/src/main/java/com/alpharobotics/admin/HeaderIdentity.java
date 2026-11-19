package com.alpharobotics.admin;
final class HeaderIdentity {
 static String label(String serverLabel,String username,boolean owner){if(serverLabel!=null&&!serverLabel.trim().isEmpty())return serverLabel.trim();if(username!=null&&!username.trim().isEmpty())return username.trim();return owner?"wssong":"관리자";}
}
