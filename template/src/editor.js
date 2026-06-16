import { Editor } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import Underline from '@tiptap/extension-underline'
import Link from '@tiptap/extension-link'
import Image from '@tiptap/extension-image'
import { Table } from '@tiptap/extension-table'
import TableRow from '@tiptap/extension-table-row'
import TableCell from '@tiptap/extension-table-cell'
import TableHeader from '@tiptap/extension-table-header'
import TextAlign from '@tiptap/extension-text-align'
import Placeholder from '@tiptap/extension-placeholder'
import CharacterCount from '@tiptap/extension-character-count'

var PRESETS = {
  basic: {
    extensions: function (opts) {
      return [
        StarterKit.configure({ heading: false, codeBlock: false, horizontalRule: false }),
        Underline,
        Link.configure({ openOnClick: false }),
        Placeholder.configure({ placeholder: opts.placeholder || '' }),
        CharacterCount
      ]
    },
    toolbar: ['bold', 'italic', 'underline', 'strike', '|', 'bulletList', 'orderedList', '|', 'link', '|', 'undo', 'redo']
  },

  standard: {
    extensions: function (opts) {
      return [
        StarterKit.configure({ heading: false, codeBlock: false }),
        Underline,
        Link.configure({ openOnClick: false }),
        Table.configure({ resizable: true }),
        TableRow,
        TableCell,
        TableHeader,
        Placeholder.configure({ placeholder: opts.placeholder || '' }),
        CharacterCount
      ]
    },
    toolbar: ['bold', 'italic', 'underline', 'strike', '|', 'bulletList', 'orderedList', '|', 'link', 'table', '|', 'undo', 'redo']
  },

  full: {
    extensions: function (opts) {
      return [
        StarterKit.configure({
          heading: { levels: [2, 3, 4] },
          codeBlock: true
        }),
        Underline,
        Link.configure({ openOnClick: false }),
        Image.configure({ allowBase64: false }),
        Table.configure({ resizable: true }),
        TableRow,
        TableCell,
        TableHeader,
        TextAlign.configure({ types: ['heading', 'paragraph'] }),
        Placeholder.configure({ placeholder: opts.placeholder || '' }),
        CharacterCount
      ]
    },
    toolbar: ['bold', 'italic', 'underline', 'strike', '|', 'heading2', 'heading3', 'heading4', '|', 'bulletList', 'orderedList', '|', 'blockquote', 'codeBlock', '|', 'alignLeft', 'alignCenter', 'alignRight', '|', 'link', 'image', 'table', '|', 'undo', 'redo']
  }
}

// SVG icon markup — these are static trusted strings, not user input
var SVG_ICONS = {
  'list-ul': '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="9" y1="6" x2="20" y2="6"/><line x1="9" y1="12" x2="20" y2="12"/><line x1="9" y1="18" x2="20" y2="18"/><circle cx="4" cy="6" r="1.5" fill="currentColor" stroke="none"/><circle cx="4" cy="12" r="1.5" fill="currentColor" stroke="none"/><circle cx="4" cy="18" r="1.5" fill="currentColor" stroke="none"/></svg>',
  'list-ol': '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="10" y1="6" x2="20" y2="6"/><line x1="10" y1="12" x2="20" y2="12"/><line x1="10" y1="18" x2="20" y2="18"/><text x="2" y="8" font-size="7" fill="currentColor" stroke="none" font-family="sans-serif">1</text><text x="2" y="14" font-size="7" fill="currentColor" stroke="none" font-family="sans-serif">2</text><text x="2" y="20" font-size="7" fill="currentColor" stroke="none" font-family="sans-serif">3</text></svg>',
  'quote': '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 21c3 0 7-1 7-8V5c0-1.25-.756-2.017-2-2H4c-1.25 0-2 .75-2 1.972V11c0 1.25.75 2 2 2 1 0 1 0 1 1v1c0 1-1 2-2 2s-1 .008-1 1.031V20c0 1 0 1 1 1z"/><path d="M15 21c3 0 7-1 7-8V5c0-1.25-.757-2.017-2-2h-4c-1.25 0-2 .75-2 1.972V11c0 1.25.75 2 2 2h.75c0 2.25.25 4-2.75 4v3c0 1 0 1 1 1z"/></svg>',
  'code': '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>',
  'link': '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>',
  'image': '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"/><circle cx="8.5" cy="8.5" r="1.5"/><polyline points="21 15 16 10 5 21"/></svg>',
  'table': '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2"/><line x1="3" y1="9" x2="21" y2="9"/><line x1="3" y1="15" x2="21" y2="15"/><line x1="9" y1="3" x2="9" y2="21"/><line x1="15" y1="3" x2="15" y2="21"/></svg>',
  'undo': '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"/></svg>',
  'redo': '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>',
  'align-left': '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="17" y1="10" x2="3" y2="10"/><line x1="21" y1="6" x2="3" y2="6"/><line x1="21" y1="14" x2="3" y2="14"/><line x1="17" y1="18" x2="3" y2="18"/></svg>',
  'align-center': '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="10" x2="6" y2="10"/><line x1="21" y1="6" x2="3" y2="6"/><line x1="21" y1="14" x2="3" y2="14"/><line x1="18" y1="18" x2="6" y2="18"/></svg>',
  'align-right': '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="21" y1="10" x2="7" y2="10"/><line x1="21" y1="6" x2="3" y2="6"/><line x1="21" y1="14" x2="3" y2="14"/><line x1="21" y1="18" x2="7" y2="18"/></svg>'
}

function createSvgButton(svgMarkup, title) {
  var btn = document.createElement('button')
  btn.type = 'button'
  btn.className = 'cm-editor-btn'
  btn.title = title
  // SVG_ICONS values are hardcoded trusted strings defined above, not user input
  var template = document.createElement('template')
  template.innerHTML = svgMarkup.trim()
  btn.appendChild(template.content.firstChild)
  return btn
}

function createTextButton(text, title) {
  var btn = document.createElement('button')
  btn.type = 'button'
  btn.className = 'cm-editor-btn'
  btn.title = title
  btn.textContent = text
  return btn
}

var BUTTON_CONFIG = {
  bold:        { icon: 'B',           svg: null, title: 'Bold',          cmd: function (e) { e.chain().focus().toggleBold().run() },        active: function (e) { return e.isActive('bold') } },
  italic:      { icon: 'I',           svg: null, title: 'Italic',        cmd: function (e) { e.chain().focus().toggleItalic().run() },      active: function (e) { return e.isActive('italic') } },
  underline:   { icon: 'U',           svg: null, title: 'Underline',     cmd: function (e) { e.chain().focus().toggleUnderline().run() },   active: function (e) { return e.isActive('underline') } },
  strike:      { icon: 'S',           svg: null, title: 'Strikethrough', cmd: function (e) { e.chain().focus().toggleStrike().run() },     active: function (e) { return e.isActive('strike') } },
  bulletList:  { icon: null,           svg: 'list-ul',  title: 'Bullet list',   cmd: function (e) { e.chain().focus().toggleBulletList().run() },  active: function (e) { return e.isActive('bulletList') } },
  orderedList: { icon: null,           svg: 'list-ol',  title: 'Numbered list', cmd: function (e) { e.chain().focus().toggleOrderedList().run() }, active: function (e) { return e.isActive('orderedList') } },
  heading2:    { icon: 'H2',          svg: null, title: 'Heading 2',     cmd: function (e) { e.chain().focus().toggleHeading({ level: 2 }).run() }, active: function (e) { return e.isActive('heading', { level: 2 }) } },
  heading3:    { icon: 'H3',          svg: null, title: 'Heading 3',     cmd: function (e) { e.chain().focus().toggleHeading({ level: 3 }).run() }, active: function (e) { return e.isActive('heading', { level: 3 }) } },
  heading4:    { icon: 'H4',          svg: null, title: 'Heading 4',     cmd: function (e) { e.chain().focus().toggleHeading({ level: 4 }).run() }, active: function (e) { return e.isActive('heading', { level: 4 }) } },
  blockquote:  { icon: null,           svg: 'quote',    title: 'Blockquote',    cmd: function (e) { e.chain().focus().toggleBlockquote().run() },  active: function (e) { return e.isActive('blockquote') } },
  codeBlock:   { icon: null,           svg: 'code',     title: 'Code block',    cmd: function (e) { e.chain().focus().toggleCodeBlock().run() },   active: function (e) { return e.isActive('codeBlock') } },
  alignLeft:   { icon: null,           svg: 'align-left',   title: 'Align left',   cmd: function (e) { e.chain().focus().setTextAlign('left').run() },   active: function (e) { return e.isActive({ textAlign: 'left' }) } },
  alignCenter: { icon: null,           svg: 'align-center', title: 'Center',       cmd: function (e) { e.chain().focus().setTextAlign('center').run() }, active: function (e) { return e.isActive({ textAlign: 'center' }) } },
  alignRight:  { icon: null,           svg: 'align-right',  title: 'Align right',  cmd: function (e) { e.chain().focus().setTextAlign('right').run() },  active: function (e) { return e.isActive({ textAlign: 'right' }) } },
  link:        { icon: null,           svg: 'link',     title: 'Link',          cmd: handleLink,  active: function (e) { return e.isActive('link') } },
  image:       { icon: null,           svg: 'image',    title: 'Image',         cmd: null },
  table:       { icon: null,           svg: 'table',    title: 'Table',         cmd: function (e) { e.chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run() } },
  undo:        { icon: null,           svg: 'undo',     title: 'Undo',          cmd: function (e) { e.chain().focus().undo().run() } },
  redo:        { icon: null,           svg: 'redo',     title: 'Redo',          cmd: function (e) { e.chain().focus().redo().run() } }
}

function handleLink(editor) {
  var prev = editor.getAttributes('link').href || ''
  var url = prompt('URL', prev)
  if (url === null) return
  if (url === '') {
    editor.chain().focus().extendMarkRange('link').unsetLink().run()
  } else {
    editor.chain().focus().extendMarkRange('link').setLink({ href: url, target: '_blank' }).run()
  }
}

function handleImage(editor, opts) {
  var uploadUrl = (opts && opts.imageUploadUrl) || '/api/images/upload'
  var input = document.createElement('input')
  input.type = 'file'
  input.accept = 'image/*'
  input.onchange = function () {
    var file = input.files[0]
    if (!file) return
    var fd = new FormData()
    fd.append('file', file)
    fetch(uploadUrl, { method: 'POST', body: fd, credentials: 'same-origin' })
      .then(function (r) { return r.json() })
      .then(function (data) {
        if (data.location) {
          editor.chain().focus().setImage({ src: data.location }).run()
        }
      })
      .catch(function (err) { console.error('Image upload failed:', err) })
  }
  input.click()
}

function buildToolbar(container, toolbarDef, editor, opts) {
  var bar = document.createElement('div')
  bar.className = 'cm-editor-toolbar'

  toolbarDef.forEach(function (name) {
    if (name === '|') {
      var sep = document.createElement('span')
      sep.className = 'cm-editor-sep'
      bar.appendChild(sep)
      return
    }
    var cfg = BUTTON_CONFIG[name]
    if (!cfg) return

    var btn
    if (cfg.svg) {
      btn = createSvgButton(SVG_ICONS[cfg.svg], cfg.title)
    } else {
      btn = createTextButton(cfg.icon || name, cfg.title)
    }

    btn.addEventListener('click', function (e) {
      e.preventDefault()
      if (name === 'image') {
        handleImage(editor, opts)
      } else if (cfg.cmd) {
        cfg.cmd(editor)
      }
    })
    bar.appendChild(btn)
  })

  container.insertBefore(bar, container.firstChild)

  editor.on('transaction', function () {
    var buttons = bar.querySelectorAll('.cm-editor-btn')
    buttons.forEach(function (btnEl) {
      var bTitle = btnEl.title
      for (var key in BUTTON_CONFIG) {
        if (BUTTON_CONFIG[key].title === bTitle && BUTTON_CONFIG[key].active) {
          if (BUTTON_CONFIG[key].active(editor)) {
            btnEl.classList.add('is-active')
          } else {
            btnEl.classList.remove('is-active')
          }
          break
        }
      }
    })
  })

  return bar
}

function createEditor(selector, options) {
  options = options || {}
  var preset = PRESETS[options.preset || 'basic']
  if (!preset) preset = PRESETS.basic

  var textarea = document.querySelector(selector)
  if (!textarea) return null

  var wrapper = document.createElement('div')
  wrapper.className = 'cm-editor-wrap'
  if (options.minHeight) wrapper.style.setProperty('--cm-editor-min-h', options.minHeight + 'px')
  if (options.maxHeight) wrapper.style.setProperty('--cm-editor-max-h', options.maxHeight + 'px')

  textarea.parentNode.insertBefore(wrapper, textarea)
  textarea.style.display = 'none'

  var contentEl = document.createElement('div')
  contentEl.className = 'cm-editor-content'
  wrapper.appendChild(contentEl)

  var editor = new Editor({
    element: contentEl,
    extensions: preset.extensions(options),
    content: textarea.value || '',
    onUpdate: function (props) {
      textarea.value = props.editor.getHTML()
      if (typeof options.onUpdate === 'function') {
        options.onUpdate(props.editor)
      }
    }
  })

  buildToolbar(wrapper, options.toolbar || preset.toolbar, editor, options)

  editor._cmTextarea = textarea
  editor._cmWrapper = wrapper

  return editor
}

function destroyEditor(editor) {
  if (!editor) return
  if (editor._cmWrapper && editor._cmWrapper.parentNode) {
    editor._cmWrapper.parentNode.removeChild(editor._cmWrapper)
  }
  if (editor._cmTextarea) {
    editor._cmTextarea.style.display = ''
  }
  editor.destroy()
}

window.cmEditor = {
  create: createEditor,
  destroy: destroyEditor,
  presets: PRESETS
}
