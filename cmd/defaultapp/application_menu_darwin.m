#import <Cocoa/Cocoa.h>

@interface DefaultAppMenuTarget : NSObject
@property NSInteger picked;
- (void)chooseApplication:(NSMenuItem *)sender;
@end
@implementation DefaultAppMenuTarget
- (void)chooseApplication:(NSMenuItem *)sender {self.picked=sender.tag;}
@end

int defaultapp_application_menu(const char *items, const char *selected, double x, double y, double width, double height) {
 @autoreleasepool {
  NSWindow *window=NSApp.keyWindow ?: NSApp.mainWindow ?: NSApp.windows.firstObject;
  NSView *view=window.contentView;
  if (!view) return -2;
  [NSApp activateIgnoringOtherApps:YES];
  [window makeKeyAndOrderFront:nil];
  NSData *data=[[NSString stringWithUTF8String:items] dataUsingEncoding:NSUTF8StringEncoding];
  NSArray *applications=[NSJSONSerialization JSONObjectWithData:data options:0 error:NULL];
  if (![applications isKindOfClass:NSArray.class]) return -3;
  NSString *selectedID=[NSString stringWithUTF8String:selected];
  NSMenu *menu=[[NSMenu alloc] initWithTitle:@"默认打开应用"];
  menu.autoenablesItems=NO;
  menu.minimumWidth=width;
  DefaultAppMenuTarget *target=[DefaultAppMenuTarget new];target.picked=-1;
  for (NSUInteger i=0;i<applications.count;i++) {
   NSDictionary *app=applications[i];
   NSMenuItem *item=[[NSMenuItem alloc] initWithTitle:app[@"name"] action:@selector(chooseApplication:) keyEquivalent:@""];
   item.target=target;item.tag=i;
   item.state=[app[@"id"] caseInsensitiveCompare:selectedID]==NSOrderedSame ? NSControlStateValueOn:NSControlStateValueOff;
   if ([app[@"path"] length]>0) {
    NSImage *icon=[[[NSWorkspace sharedWorkspace] iconForFile:app[@"path"]] copy];
    icon.size=NSMakeSize(18,18);item.image=icon;
   }
   [menu addItem:item];
  }
  CGFloat anchorY=view.isFlipped ? y+height : view.bounds.size.height-y-height;
  [menu popUpMenuPositioningItem:nil atLocation:NSMakePoint(x,anchorY) inView:view];
  return (int)target.picked;
 }
}
