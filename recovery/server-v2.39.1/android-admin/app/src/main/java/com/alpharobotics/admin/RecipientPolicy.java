package com.alpharobotics.admin;
/** Device-local defense against a delayed push for a previous account/role. */
final class RecipientPolicy {
 static boolean accepts(boolean signedOut,boolean off,String role,String account,String kind,String recipient){
  if(signedOut||off)return false;
  boolean purchase="purchasing".equals(role),call="purchasing_call".equals(kind);
  if(purchase!=call)return false;
  return recipient==null?!purchase:recipient.equals(account);
 }
}
