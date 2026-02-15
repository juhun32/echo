import type { ChatSession } from './chat';

export let chatSessions: ChatSession[] = [
	{
		id: 'chat-1',
		title: 'How to center a div',
		messages: [
			{ id: 'msg-1', role: 'user', text: 'How do I center a div?' },
			{
				id: 'msg-2',
				role: 'assistant',
				text: `Centering a \`div\` element is a very common task in web development. There are several effective methods, and the best one to use often depends on the context.

### 1. Using \`margin: auto\`
This is the classic method for centering a block-level element horizontally.
* **Requirement:** The \`div\` must have a defined \`width\`.
* **How it works:** \`margin: auto\` distributes remaining horizontal space equally.

**CSS Example:**
\`.centered-div { width: 50%; margin: 0 auto; }\`

---

### 2. Using Flexbox
Flexbox makes centering (both horizontally and vertically) incredibly easy.

* **How it works:** Apply properties to the parent container.
* **Properties:**
    * \`display: flex;\`
    * \`justify-content: center;\` (Horizontal)
    * \`align-items: center;\` (Vertical)

---

### 3. Using CSS Grid
CSS Grid is a powerful system that excels at centering with very little code.

* **How it works:** Use the \`place-items\` shorthand on the parent.
* **Properties:**
    * \`display: grid;\`
    * \`place-items: center;\`

---

### 4. Absolute Positioning and Transform
Useful for centering an element that is removed from the normal document flow.

* **Requirement:** Parent must have \`position: relative;\`.
* **How it works:**
    1. Set child to \`position: absolute; top: 50%; left: 50%;\`.
    2. Use \`transform: translate(-50%, -50%);\` to shift it back to the true center.

---

### Summary: Which one to choose?
* **Simple horizontal:** \`margin: auto\`.
* **Modern layouts:** **Flexbox** or **Grid** (Recommended).
* **Overlays/Modals:** **Absolute Positioning**.

**Note:** \`text-align: center;\` centers **inline** content (text/images) but not the \`div\` container itself.`
			}
		]
	}
];
