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


