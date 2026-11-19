package com.alpharobotics.admin;
/** Process-local screen access requires explicit authentication; a saved server login is not a screen unlock. */
final class UnlockPolicy {
 private boolean open;
 boolean isOpen(){return open;}
 void lock(){open=false;}
 void background(boolean choosingFile){if(!choosingFile)lock();}
 void authenticated(){open=true;}
}
