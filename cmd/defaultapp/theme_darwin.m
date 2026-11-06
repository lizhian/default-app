#import <Cocoa/Cocoa.h>

// Resolve AppKit tokens in Aqua so text and surfaces share one appearance.
void defaultapp_theme_color(int index, float *rgba) {
 @autoreleasepool {
  [[NSAppearance appearanceNamed:NSAppearanceNameAqua] performAsCurrentDrawingAppearance:^{
   NSColor *colours[]={
    NSColor.windowBackgroundColor,
    NSColor.labelColor,
    NSColor.textBackgroundColor,
    [NSColor.windowBackgroundColor blendedColorWithFraction:0.4 ofColor:NSColor.textBackgroundColor],
    NSColor.secondaryLabelColor,
    NSColor.separatorColor,
    NSColor.controlAccentColor,
    NSColor.alternateSelectedControlTextColor,
    [NSColor.controlAccentColor blendedColorWithFraction:0.9 ofColor:NSColor.textBackgroundColor],
    NSColor.systemGreenColor
   };
   NSColor *colour=[colours[index] colorUsingColorSpace:NSColorSpace.sRGBColorSpace];
   rgba[0]=colour.redComponent;
   rgba[1]=colour.greenComponent;
   rgba[2]=colour.blueComponent;
   rgba[3]=colour.alphaComponent;
  }];
 }
}
