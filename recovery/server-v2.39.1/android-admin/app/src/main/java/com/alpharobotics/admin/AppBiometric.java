package com.alpharobotics.admin;

import android.app.Activity;
import android.app.AlertDialog;
import android.content.Context;
import android.hardware.fingerprint.FingerprintManager;
import android.os.Build;
import android.os.CancellationSignal;
import android.hardware.biometrics.BiometricPrompt;

/** Uses Android's biometric verification; never opens a browser or receives fingerprints. */
final class AppBiometric {
 interface Result {void success();void password();void canceled(String message);}
 private int generation;
 private CancellationSignal cancellation;
 private AlertDialog legacyDialog;
 static boolean available(Context context){try{if(Build.VERSION.SDK_INT>=30){android.hardware.biometrics.BiometricManager m=context.getSystemService(android.hardware.biometrics.BiometricManager.class);return m!=null&&m.canAuthenticate(android.hardware.biometrics.BiometricManager.Authenticators.BIOMETRIC_STRONG)==android.hardware.biometrics.BiometricManager.BIOMETRIC_SUCCESS;}FingerprintManager f=context.getSystemService(FingerprintManager.class);return f!=null&&f.isHardwareDetected()&&f.hasEnrolledFingerprints();}catch(Exception e){return false;}}
 void cancel(){generation++;if(cancellation!=null)cancellation.cancel();cancellation=null;if(legacyDialog!=null){legacyDialog.dismiss();legacyDialog=null;}}
 void authenticate(Activity activity,Result result){
  cancel();if(!available(activity)){result.canceled("휴대폰에 등록된 지문이 없습니다. 비밀번호로 로그인해 주세요.");return;}
  final int request=generation;cancellation=new CancellationSignal();
  if(Build.VERSION.SDK_INT>=28){
   BiometricPrompt.Builder builder=new BiometricPrompt.Builder(activity).setTitle("알파 관리자").setSubtitle("휴대폰 지문으로 앱을 열어 주세요.").setNegativeButton("비밀번호로 로그인",activity.getMainExecutor(),(d,n)->{if(request!=generation)return;cancel();result.password();});
   if(Build.VERSION.SDK_INT>=30)builder.setAllowedAuthenticators(android.hardware.biometrics.BiometricManager.Authenticators.BIOMETRIC_STRONG);
   builder.build().authenticate(cancellation,activity.getMainExecutor(),new BiometricPrompt.AuthenticationCallback(){
    @Override public void onAuthenticationSucceeded(BiometricPrompt.AuthenticationResult r){if(request!=generation)return;generation++;cancellation=null;result.success();}
    @Override public void onAuthenticationError(int code,CharSequence message){if(request!=generation)return;generation++;cancellation=null;if(code==13){result.password();return;}result.canceled(message==null?"지문 인증을 다시 시도해 주세요.":message.toString());}
   });
  }else{
   legacyDialog=new AlertDialog.Builder(activity).setTitle("알파 관리자 지문 확인").setMessage("휴대폰 지문 센서에 손가락을 대 주세요.").setNegativeButton("비밀번호로 로그인",(d,n)->{if(request!=generation)return;cancel();result.password();}).setOnCancelListener(d->{if(request!=generation)return;cancel();result.canceled("지문 인증을 취소했습니다.");}).create();legacyDialog.show();
   activity.getSystemService(FingerprintManager.class).authenticate(null,cancellation,0,new FingerprintManager.AuthenticationCallback(){
    @Override public void onAuthenticationSucceeded(FingerprintManager.AuthenticationResult r){if(request!=generation)return;generation++;cancellation=null;if(legacyDialog!=null){legacyDialog.dismiss();legacyDialog=null;}result.success();}
    @Override public void onAuthenticationError(int code,CharSequence message){if(request!=generation)return;generation++;if(legacyDialog!=null){legacyDialog.dismiss();legacyDialog=null;}cancellation=null;result.canceled(message==null?"지문 인증을 다시 시도해 주세요.":message.toString());}
   },null);
  }
 }
}
