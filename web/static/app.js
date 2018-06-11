const host = 'localhost:5862';
const TARGET_ALL = 'target_all_components';

const modelist = ace.require("ace/ext/modelist")

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
    props: ['editorId', 'content', 'lang', 'theme', "filename"],
    data: function () {
        return {
            editor: Object,
            beforeContent: '',
            mode: modelist.getModeForPath("blank.json")
        }
    },
    watch: {
        'content': function (value) {
            if (this.beforeContent !== value) {
                this.editor.setValue(value, 1);
            }
        },
        "filename": function (value) {
            this.mode = modelist.getModeForPath(value).mode;
            this.editor.session.setMode(this.mode);
        }
    },
    methods: {
        clear: function () {
            this.editor.setValue("", 1);
        }
    },
    mounted: function () {
        this.editor = window.ace.edit(this.editorId);
        this.editor.setValue(this.content, 1);
        this.editor.session.setMode(this.mode.mode);

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


Vue.component("tab", {
    delimiters: ["[[", "]]"],
    template: `
<li class="tab-item" v-bind:class="{ active: active }">
    <a v-on:click="load" href="#"><span class="btn btn-clear" v-on:click="close" v-bind:active="active"></span> [[ name ]]</a>
</li>`,
    props: ["name", "active"],
    data: function () {
        return {
            data: "",
        }
    },
    methods: {
        load: function () {
            let request = new XMLHttpRequest();
            let self = this;
            request.addEventListener("load", function (ev) {
                if (ev.currentTarget.status === 200) {
                    self.data = JSON.parse(ev.currentTarget.responseText).data;
                    self.$emit("load", self.data, self.$props.name)
                }
            });
            request.open("GET", "/api/file/" + this.$props.name);
            request.setRequestHeader("Content-Type", "text/plain");
            request.send();
        },
        close: function () {
            this.$emit("close", this)
        }
    },
    mounted: function () {
        this.load()
    }
});


let protocol = 'ws://';
if (window.location.protocol === 'https:') {
    protocol = 'wss://';
}

// construct the url for the web socket to listen on
let webSocketUrl = protocol + host;

const socket = new WebSocket(webSocketUrl + "/ws/statistics");

const statisticsEmitter = new Vue();

socket.onmessage = function (data) {
    statisticsEmitter.$emit("receive", JSON.parse(data.data));
};


let vm = new Vue({
    el: "#vue",
    delimiters: ["[[", "]]"],
    data: function () {
        return {
            editing: [],
            openFile: "",
            file: {},
            input: `{}`,
            queryPath: "",
            statistics: {"CPU": [0.0], "Memory": 0.0}
        }
    },
    methods: {
        updateInput: function (val, filename) {
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
            let self = this;
            _.debounce(function () {
                self.openFile = "";
                self.input = "";
            }, 50)()
        },
        loadTab: function (data, name) {
            // Set the active state for the selected tab.
            for (let i = 0; i < this.editing.length; i++) {
                this.editing[i].active = this.editing[i].name === name;
            }
            this.openFile = name;
            this.input = data;
        },
        addTab: function (name) {
            // Reset all active states.
            for (let i = 0; i < this.editing.length; i++) {
                this.editing[i].active = false;
            }
            this.openFile = name;
            // Set the active state for the new tab.
            this.editing.push({name: name, active: true})
        },
        newTab: function () {
            let name = prompt("Name the new file");
            this.addTab(name);
            this.openFile = name;
            this.input = "";
        },
        runPipeline: function () {
            let request = new XMLHttpRequest();
            let self = this;
            request.addEventListener("load", function (ev) {
                if (ev.currentTarget.status === 200) {
                    self.updateFiles();
                } else {
                    alert(ev.currentTarget.responseText)
                }
            });
            request.open("POST", "/api/run");
            request.setRequestHeader("Content-Type", "application/x-www-form-urlencoded");
            request.send("pipeline=" + this.input + "&path=" + self.queryPath);
        },
        stopPipeline: function () {
            alert("not yet implemented, server restart required");
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
            request.send();
        },
        saveFile: function () {
            let request = new XMLHttpRequest();
            let self = this;
            request.addEventListener("load", function (ev) {
                if (ev.currentTarget.status === 200) {
                    self.updateFiles();
                } else {
                    alert(ev.currentTarget.responseText)
                }
            });
            request.open("POST", "/api/save/" + this.openFile);
            request.setRequestHeader("Content-Type", "application/x-www-form-urlencoded");
            // noinspection JSCheckFunctionSignatures
            request.send("content=" + this.input);
        },
        handleReceive: function (data) {
            this.statistics = data;
        },
        formatFloat: function (n) {
            return n.toFixed(2);
        }
    },
    mounted: function () {
        this.updateFiles()
    },
    created: function () {
        statisticsEmitter.$on("receive", this.handleReceive)
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