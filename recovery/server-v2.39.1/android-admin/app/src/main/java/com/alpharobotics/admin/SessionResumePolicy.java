package com.alpharobotics.admin;
/** Saved capability permits a server-verified restore, never a cached unlock. */
final class SessionResumePolicy {
 static boolean shouldRestore(boolean foreground,boolean open,boolean busy,boolean signedOut,String grant,String secret){
  return foreground&&!open&&!busy&&!signedOut&&grant!=null&&!grant.isEmpty()&&secret!=null&&!secret.isEmpty();
 }
}
