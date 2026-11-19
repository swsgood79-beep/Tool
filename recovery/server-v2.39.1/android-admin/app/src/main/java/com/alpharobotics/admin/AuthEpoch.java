package com.alpharobotics.admin;
/** Invalidates asynchronous UI/login work after explicit logout or an account change. */
final class AuthEpoch {
 private long value;
 synchronized long current(){return value;}
 synchronized long invalidate(){return ++value;}
 synchronized boolean accepts(long snapshot){return snapshot==value;}
}
