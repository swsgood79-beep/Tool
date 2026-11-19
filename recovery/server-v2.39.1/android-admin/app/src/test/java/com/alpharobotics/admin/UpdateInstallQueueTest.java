package com.alpharobotics.admin;
import org.junit.Test;
import static org.junit.Assert.*;
public class UpdateInstallQueueTest {
 @Test public void acceptedInstallSurvivesLoginAndWorkBusy(){UpdateInstallQueue q=new UpdateInstallQueue();int[] calls={0};q.offer(()->calls[0]++);assertNull(q.takeIfReady(true,true));assertNotNull(q.peek());assertNull(q.takeIfReady(true,true));q.takeIfReady(true,false).run();assertEquals(1,calls[0]);assertNull(q.takeIfReady(true,false));}
 @Test public void backgroundDoesNotLaunchInstallerAndResumeRunsOnce(){UpdateInstallQueue q=new UpdateInstallQueue();int[] calls={0};q.offer(()->calls[0]++);assertNull(q.takeIfReady(false,false));assertNotNull(q.peek());q.takeIfReady(true,false).run();assertEquals(1,calls[0]);assertNull(q.peek());}
 @Test public void replacementAndDestroyedActivityDoNotRunStaleActions(){UpdateInstallQueue q=new UpdateInstallQueue();Runnable old=()->fail("stale action");q.offer(old);Runnable current=()->{};q.offer(current);assertNotSame(old,q.peek());assertSame(current,q.takeIfReady(true,false));q.offer(old);q.clear();assertNull(q.takeIfReady(true,false));}
}
