package com.alpharobotics.admin;

/** Preserve an accepted installation while login or work is busy. UI thread only. */
final class UpdateInstallQueue {
 private Runnable pending;
 void offer(Runnable action){pending=action;}
 Runnable peek(){return pending;}
 Runnable takeIfReady(boolean foreground,boolean busy){if(!foreground||busy)return null;Runnable value=pending;pending=null;return value;}
 void clear(){pending=null;}
}
