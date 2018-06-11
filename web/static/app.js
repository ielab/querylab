const TARGET_ALL = 'target_all_components';

function dispatch(component, event, data) {
    const parent = component.$parent;

    if (parent) {
        parent.$emit(event, data);
        dispatch(parent, event, data);
    }
}

// Editor component.
Vue.component('editor', {
    template: '<div :id="editorId" class="editor"></div>',
    props: ['editorId', 'content', 'lang', 'theme'],
    data: function () {
        return {
            editor: Object,
            beforeContent: ''
        }
    },
    watch: {
        'content': function (value) {
            if (this.beforeContent !== value) {
                this.editor.setValue(value, 1)
            }
        }
    },
    mounted: function () {
        this.editor = window.ace.edit(this.editorId);
        this.editor.setValue(this.content, 1);
        this.editor.session.setMode("ace/mode/json");

        let self = this;
        this.editor.on('change', function () {
            self.beforeContent = self.editor.getValue();
            self.$emit('change-content', self.editor.getValue());
        });
    }
});

// Side menu files component.
let fileId = 0;
Vue.component("file", {
    delimiters: ["[[", "]]"],
    template: "#file",
    props: ["file"],
    data: function () {
        return {
            id: fileId++
        }
    },
    methods: {
        load: function () {
            let path = this.file.path.join("/") + "/" + this.file.name;
            if (this.file.path.length === 0) {
                path = this.file.name;
            }
            dispatch(this, "load", path);
            // this.$parent.$emit("load", path);
        }
    }
});


let tabId = 0;
Vue.component("tab", {
    delimiters: ["[[", "]]"],
    template: `
<li class="tab-item">
    <a v-on:click="load" href="#"><span class="btn btn-clear" v-on:click="close" v-bind:active="active"></span> [[name]]</a>
</li>`,
    props: ["name"],
    data: function () {
        return {
            id: tabId++,
            active: false,
            data: ""
        }
    },
    methods: {
        load: function () {
            this.active = true;
            this.$emit("load", this.data, this.$props.name);
        },
        close: function () {
            this.$emit("close", this)
        }
    },
    mounted: function () {
        let request = new XMLHttpRequest();
        let self = this;
        request.addEventListener("load", function (ev) {
            if (ev.currentTarget.status === 200) {
                self.data = JSON.parse(ev.currentTarget.responseText).data;
                self.$emit("load", self.data)
            }
        });
        request.open("GET", "/api/file/" + this.name);
        request.setRequestHeader("Content-Type", "text/plain");
        request.send()
    }
});

let vm = new Vue({
    el: "#vue",
    data: {
        editing: [],
        openFile: "",
        file: {},
        input: `{}`,
        output: "",
        queryPath: "",
    },
    methods: {
        updateInput: function (val) {
            let self = this;
            // _.debounce(function (e) {
            self.input = val;
            self.$nextTick(function () {
                // self.transform()
            });
            // }, 500)()
        },
        closeTab: function (e) {
            this.editing = _.without(this.editing, _.find(this.editing, {
                name: e.name
            }));
        },
        loadTab: function (data, name) {
            this.input = data;
            this.openFile = name;
        },
        addTab: function (name) {
            this.editing.push({name: name})
        },
        newTab: function () {
            let name = prompt("Name the new file");
            this.addTab(name)
        },
        runPipeline: function () {
            let request = new XMLHttpRequest();
            let self = this;
            request.addEventListener("load", function (ev) {
                if (ev.currentTarget.status === 200) {
                    self.output = JSON.parse(ev.currentTarget.responseText)
                } else {
                    alert(ev.currentTarget.responseText)
                }
            });
            request.open("POST", "/api/run");
            request.setRequestHeader("Content-Type", "application/x-www-form-urlencoded");
            request.send("pipeline=" + this.input + "&path=" + self.queryPath);
        },
        stopPipeline: function () {

        },
        updateFiles: function () {
            let request = new XMLHttpRequest();
            let self = this;
            request.addEventListener("load", function (ev) {
                if (ev.currentTarget.status === 200) {
                    self.file = JSON.parse(ev.currentTarget.responseText)
                }
            });
            request.open("GET", "/api/files");
            request.setRequestHeader("Content-Type", "application/json");
            request.send(this.input);
        },
        saveFile: function () {
            let request = new XMLHttpRequest();
            let self = this;
            request.addEventListener("load", function (ev) {
                if (ev.currentTarget.status === 200) {
                    self.output = JSON.parse(ev.currentTarget.responseText)
                } else {
                    alert(ev.currentTarget.responseText)
                }
            });
            request.open("POST", "/api/save/" + this.openFile);
            request.setRequestHeader("Content-Type", "application/x-www-form-urlencoded");
            request.send("content=" + this.input);
        }
    },
    mounted: function () {
        this.updateFiles()
    }
});

document.addEventListener("keydown", function () {
    if ((event.which === 115 || event.which === 83) && (event.ctrlKey || event.metaKey) || (event.which === 19)) {
        event.preventDefault();
        vm.saveFile();
        return false;
    }
    return true;
});

// window.onload = function () {
document.getElementById("loading").remove();
// }