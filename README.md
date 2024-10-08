# go-plugin-engine
This is a plugin engine/manager that uses the go Extism project SDK and allows a Go application to load this module as a wasm/wasi plugin (via extism as well) which can then be used to find/load other Extism project PDK wasm/wasi modules as plugins. 


Extension point:
  A menu has an extension point (in a GUI for example) that other plugins can provide extensions for that add menus and menu items.
  A menu item when clicked would call the menu item plugin function possibly with some data (such as context specific data.. e.g. what window is on top, any selection data, etc).
  The engine also supports events. Events would be used in a situation such as context aware windows that work with menus and icons (menu buttons). For example, a copy/paste set of menu
  items would only make sense to "work" (or visually be enabled) when something that can be copied is selected. A text editor plugin could fire an event "selected" with some metadata that
  allows the copy and/or paste menuitem to highlight/enable. Whatever it was pasted in to would then fire a "selection pasted" event so that any listeners can "disable" or turn off the icon
  or menu item from being clickable again. This goes a step further in that.. what happens when a text editor plugin has a selection and enables the copy button, but the user then selects
  a different window where nothing is selected? The copy item should disable.. as nothing is selected in the current "top" window. So for a really GOOD GUI to work right, things like windows,
  dialogs, etc should properly fire the appropriate events to allow for the GUI to update appropriately based on context awareness. If a FILE window is selected, the text editor loses focus
  and like the selection turns gray.. to show the text editor is not in focus (but selection still exists). If the user goes BACK to the text editor window, it should re-enable the copy
  button/icon and show the selection.. saving the selection state between the context window switch. If the user selects a file in the left hand file list window, the copy is enabled again,
  and if they go back to the text editor, the switch to the text editor window might first disable, then see a selection exists so re-enable the copy option.  When the COPY is selected, it
  would know the plugin owner of the selected content, ask that plugin to "copy" the data however it needs to and disable the copy and enable the paste button. If while content is "in memory"
  the user switches to the file panel, the plugin (text editor) is STILL holding the state of the selection however it needs to, but now the user can see that the Paste option is disabled
  and then they select a file and now the COPY item becomes enabled. As they switch between the windows each window would fire "state" such as "selection" and/or other events so that context
  aware changes occur accordingly. This of course amplifies for the number of plugins that are also listening to the same events. E.g. there could be windows NOT on top or not even selectable
(e.g a status bar at the bottom that just shows status text) that could ALSO be listening to events and updating. It does not have to be a top selected window to make changes.

  The above portrays a GUI application frame/system. Events/etc could be for non GUI non visual use cases as well. You can certainly have a server side application that is built from plugins
  and can add events, respond to events, etc just the same. 



Extension:
  Extensions are how plugins can add contributions to plugins that provide ExtensionPoint's. Extensions are the implementation (typically) to the ExtensionPoint's contract (interface, etc).
  Plugin developers would determine the ExtensionPoint(s) to be contributed to and follow any details regarding the structure an ExtensionPoint expects to pass to the Extension, and any return
  structure. This is ALWAYS in the form of a []byte and it is up to the developer of the plugin Extension to ensure proper marshal and unmarshal of data at both ends. Hopefully ExtensionPoint 
  developers provide plenty of details with regards to the purpose of the ExtensionPoint, the structures expected as parameters and return values, and so on.




EventListener and Event:
  The engine supports the ability for plugins to register listeners, which are functions that take in an event string value, and a data []byte. It is up to the sender of the event to marshal
  the event structure into a []byte to send it. It is up to each listener to reverse that process.. unmarshal the json back into the appropriate structure to utilize the data within.
  Plugins can call the provided host function SendEvent using the PluginEngine PDK for their specific language. The PDK wraps several host functions to abstract away the Extism PDK particulars
  for setting up parameters correctly to be passed to the host function and any return value. 

DEPENDENCY:
 There are two forms of dependencies. One is where a plugin can NOT function without the other plugin being resolved/available. The other
is more of "discovery" in that a plugin can look up a given other plugin's extension point(s) and if they are available, can make use of
them (to call extensions of those extension points).

NOTES:
When a plugin extension code executes.. it can look up (discover) extension points of other plugins. If the extension 
DEPENDS on a given other plugins extension point, it adds that extnesion point to its Plugin dependency list. This is to ensure
that THIS plugin can NOT be called/work/resolved UNLESS a dependent plugin AND extension point is resolved/available. In the case of
dynamic discovery, however, there is no dependency needed. It is up to the plugin extension code to NOT FAIL IF a plugin
extension point does not exist (e.g. not resolved for any reason). 
Plugin extension code uses the plugin engine framework to call upon another extension point function or extension. This 
is to ensure all calls go thru the engine.