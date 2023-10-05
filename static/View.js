'use strict';


// 28, 14, 97.6, 42.9, 161.0, 162.8
function Grid(width = 28, height = 14, pixelWidth = 97.6, spaceWidth = 42.9, pixelHeight = 161.0, spaceHeight = 162.8) {
	this.width  = width;
	this.height = height;
	this.pX = pixelWidth;
	this.sX = spaceWidth;
	this.pY = pixelHeight;
	this.sY = spaceHeight;
}

Grid.prototype.createDOMElements = function(width, height) {
	width  = width  || window.innerWidth;
	height = height || window.innerHeight;
	
	// the total pixel fragment size
	var totalX = this.pX + this.sX;
	var totalY = this.pY + this.sY;
	
	// the perfect factor to fill the area
	var factor = Math.min(
			width  / this.width / totalX,
			height / this.height / totalY);
	
	var canvasWidth  = factor * this.width  * totalX;
	var canvasHeight = factor * this.height * totalY;
	
	// Create DOM elements
	let div = document.createElement('div');
	div.classList.add('view');
	//div.style.position = 'relative';
	
	// background
	let bg = document.createElement('canvas');
	bg.setAttribute('width',  canvasWidth);
	bg.setAttribute('height', canvasHeight);
	//bg.style.position = 'absolute';
	//bg.style.top  = 0;
	//bg.style.left = 0;
	this.backgroundContext = bg.getContext('2d');
	this.renderBg();
	
	// main
	let main = document.createElement('canvas');
	main.setAttribute('width',  canvasWidth);
	main.setAttribute('height', canvasHeight);
	//main.style.position = 'absolute';
	//main.style.top  = 0;
	//main.style.left = 0;
	this.context    = main.getContext('2d');
	
	// overlay
	let overlay = document.createElement('canvas');
	overlay.setAttribute('width',  canvasWidth);
	overlay.setAttribute('height', canvasHeight);
	//overlay.style.position = 'absolute';
	//overlay.style.top  = 0;
	//overlay.style.left = 0;
	this.overlayContext = overlay.getContext('2d');
	
	// assemble the elements
	div.appendChild(bg);
	div.appendChild(main);
	div.appendChild(overlay);
	
	return div;
}

Grid.prototype.updateRenderRatios = function(ctx) {
	if (ctx == undefined && this.context == undefined) {
		return;
	}
	ctx = ctx || this.context;
	if (this.context == undefined) {
		this.context = ctx;
	}
	
	// get: the canvas size
	var canvasWidth = ctx.canvas.width;
	var canvasHeight = ctx.canvas.height;
	
	// the total pixel fragment size
	var totalX = this.pX + this.sX;
	var totalY = this.pY + this.sY;
	
	// the perfect factor to fill the canvas
	var factor = Math.min(
			canvasWidth  / this.width / totalX,
			canvasHeight / this.height / totalY);
	
	// save the ratios
	this.ratio = {
		factor: factor,
		totalX: totalX,
		totalY: totalY,
		// calculate actual pixel and frame sizes
		pX: this.pX     * factor,
		pY: this.pY     * factor,
		sX: this.sX     * factor,
		sY: this.sY     * factor,
		stepX: totalX   * factor,
		stepY: totalY   * factor,
		// the border offset
		startX: this.sX * factor / 2,
		startY: this.sY * factor / 2
	};
	return this.ratio;
}

Grid.prototype.getPos = function(mouse,fuzzy) {
	if (fuzzy == undefined) fuzzy = false;
	var ratio = this.ratio || this.updateRenderRatios(ctx);
	
	// mouse without border offset
	var currX = mouse.x - ratio.startX;
	var currY = mouse.y - ratio.startY;
	
	// on fuzzy position take half of the space as an additional border
	if (fuzzy) {
		currX += ratio.pX/2;
		currY += ratio.pY/2;
	}
	
	// get the position
	var posX = Math.floor(currX / ratio.stepX);
	var posY = Math.floor(currY / ratio.stepY);
	
	if (!fuzzy) {
		// get the position inside the pixel frame
		currX -= posX * ratio.stepX;
		currY -= posY * ratio.stepY;
		
		if (posX<0 || posY<0 || posX>=this.width || posY>=this.height
				|| currX>ratio.pX || currY>ratio.pY) {
			return undefined;
		}
	} else if (posX<0 || posY<0 || posX>=this.width || posY>=this.height) {
		return undefined;
	}
	
	return {x: posX, y: posY}
}

Grid.prototype.renderBg = function(ctx) {
	var ctx = ctx || this.backgroundContext;
	var ratio = this.ratio || this.updateRenderRatios(ctx);
	
	ctx.fillStyle = '#000000';
	ctx.clearRect(0, 0, ctx.canvas.width, ctx.canvas.height);
	ctx.fillStyle = 'rgb(0,0,0)';
	ctx.fillRect(Math.floor(ratio.sX / 4),
				Math.floor(ratio.sY / 4),
				Math.floor(ratio.stepX*this.width - ratio.sX / 4),
				Math.floor(ratio.stepY*this.height - ratio.sY / 4));
	
	ctx.lineCap = "butt";
	ctx.strokeStyle = '#888888';
	ctx.lineWidth = Math.round(ratio.sX / 2);
	
	// draw lines between the columns
	var currY = ratio.sY / 4;
	var currX = ratio.stepX;
	ctx.beginPath();
	var lineHeight = Math.floor(currY + (this.height-1) * ratio.stepY + ratio.pY + ratio.sY / 2);
	// do one less (x = 1)
	for (let x = 1; x<this.width; x++, currX += ratio.stepX) {
		ctx.moveTo(Math.floor(currX), Math.floor(currY));
		ctx.lineTo(Math.floor(currX), lineHeight);
	}
	ctx.stroke();
	
	/* no transparent bg
	currY = ratio.startY;
	for (let y = 0; y<this.height; y++, currY += ratio.stepY) {
		currX = ratio.startX;
		for (let x = 0; x<this.width; x++, currX += ratio.stepX) {
			
			// as transparent background create a grey pattern
			
			if(this.style) {
				ctx.fillStyle = 'rgb(255,255,255)';
				ctx.fillRect(
					Math.floor(currX),
					Math.floor(currY),
					Math.floor(currX+ratio.pX)-Math.floor(currX),
					Math.floor(currY+ratio.pY)-Math.floor(currY));
				
				ctx.fillStyle = 'rgb(191,191,191)';
				ctx.fillRect(
						Math.floor(currX),
						Math.floor(currY),
						Math.floor(ratio.pX/2),
						Math.floor(ratio.pY/2));
				ctx.fillRect(
						Math.floor(currX) + Math.floor(ratio.pX/2),
						Math.floor(currY) + Math.floor(ratio.pY/2),
						Math.floor(ratio.pX/2),
						Math.floor(ratio.pY/2));
			} else {
				ctx.fillStyle = 'rgb(191,191,191)';
				ctx.fillRect(
						Math.floor(currX),
						Math.floor(currY),
						Math.floor(ratio.pX),
						Math.floor(ratio.pY));
				
				ctx.fillStyle = 'rgb(255,255,255)';
				ctx.fillRect(
						Math.floor(currX),
						Math.floor(currY),
						Math.floor(ratio.pX/2),
						Math.floor(ratio.pY/2));
				ctx.fillRect(
						Math.floor(currX) + Math.floor(ratio.pX/2),
						Math.floor(currY) + Math.floor(ratio.pY/2),
						Math.floor(ratio.pX/2),
						Math.floor(ratio.pY/2));
			}
		}
	}
	*/
}

Grid.prototype.renderImg = function(img, ctx) {
	var ctx = ctx || this.context;
	var ratio = this.ratio || this.updateRenderRatios(ctx);
	
	//ctx.clearRect(mouse.lastX-30, mouse.lastY-30,60,60);
	ctx.clearRect(0,0,ctx.canvas.width,ctx.canvas.height);
	
	if (!img) {
		return;
	}
	
	let currY = ratio.startY;
	let i = 0;
	for (let y = 0; y<this.height; y++, currY += ratio.stepY) {
		let currX = ratio.startX;
		for (let x = 0; x<this.width; x++, currX += ratio.stepX) {
			ctx.fillStyle = 'rgb('+img[i++]+','+img[i++]+','+img[i++]+')';
			ctx.fillRect(Math.floor(currX), Math.floor(currY), Math.floor(ratio.pX), Math.floor(ratio.pY));
		}
	}
}

Grid.prototype.renderOverlay = function(mouse, ctx) {
	var ctx = ctx || this.overlayContext;
	var ratio = this.ratio || this.updateRenderRatios(ctx);
	
	ctx.clearRect(
			mouse.lastX-ratio.stepX-10, mouse.lastY-ratio.stepY-10,
			2*ratio.stepX + 20, 2*ratio.stepY + 20);
	
	var pos = this.getPos(mouse, true);
	
	if (pos != undefined) {	
		let currX = ratio.sX / 2 + pos.x * ratio.stepX;
		let currY = ratio.sY / 2 + pos.y * ratio.stepY;
		
		ctx.strokeStyle = "yellow";
		ctx.strokeRect(Math.floor(currX), Math.floor(currY), Math.floor(ratio.pX), Math.floor(ratio.pY));
	}
	
	ctx.lineCap = "round";
	ctx.strokeStyle = 'rgb(255,0,0)';
	ctx.lineWidth = 2;
	
	ctx.beginPath();
	ctx.moveTo(mouse.x, mouse.y);
	ctx.lineTo(mouse.x, mouse.y);
	ctx.stroke();
	
	mouse.lastX = mouse.x;
	mouse.lastY = mouse.y;
}









let page_is_open = true;
window.addEventListener("beforeunload", () => {console.log('beforeunload');page_is_open = false;} );

function LHWebSocket(auth, onopen) {
	this.auth = auth;
	
	this.next_StreamID    = 0;
	this.stream_handler   = [];
	this.streammap        = {};
	this.last_stream_data = []
	
	this.next_RequestID   = -1;
	this.free_RequestIDs  = [];
	this.request_handler  = [];
	
	this.onclose = null;
	this.closed = true;
	
	this.onopen = onopen;
	
	this.reopen();

}


LHWebSocket.prototype.msgpack_options = {codec: msgpack.createCodec({binarraybuffer: true})};

LHWebSocket.prototype.reopen = function() {
	if (!this.closed) return;
	
	this.closed = false;
	
	// '/user/'+auth.USER+'/model'
	const ws = new WebSocket('wss://lighthouse.uni-kiel.de/websocket');
	this.ws = ws;
	ws.binaryType = "arraybuffer";
	
	const _this = this;
	
	this.openPromise = new Promise((resolve, reject) => {
		ws.onopen = (e) => {
			_this.request_handler.forEach(request => {
				_this.sendObj(request.packet);
				console.log('Resending Request');
			});
			const streammap = _this.streammap;
			for (let path in streammap) {
				if (!streammap.hasOwnProperty(path)) continue;
				const stream_info = streammap[path];
				const stream_handler = _this.stream_handler[stream_info.reid];
				if (stream_handler.handlers.find(h=>h.active) !== undefined) {
					console.log('Reopening STREAM '+path);
					_this.sendObj({
						VERB: 'STREAM',
						PATH: path.split('/'),
						AUTH: this.auth,
						META: {},
						PAYL: null,
						REID: stream_info.reid
					});
					stream_info.active = true;
					stream_handler.first = true;
				} else {
					stream_info.active = false;
				}
			}
			if (typeof _this.onopen === 'function') _this.onopen(_this);
			_this.onopen = null;
			resolve(_this);
		}
	});
	
	ws.onmessage = (event) => {
		try {
			const packet = msgpack.decode(new Uint8Array(event.data), this.msgpack_options);
			//if(packet.REID<0) console.log('recv: ', packet.REID, packet.PAYL);
			if (packet.RNUM !== 200) {
				console.error('Websocket Error: "'+packet.RNUM+' '+packet.RESPONSE+'" on '+
							(packet.REID<0 ? 'Request' : 'Stream'));
				return;
			}
			if (packet.REID<0) { // Request
				this.free_RequestIDs.push(packet.REID);
				const handler = this.request_handler[-packet.REID];
				this.request_handler[-packet.REID] = null;
				if (handler) {
					if (packet.RNUM === 200) {
						handler.resolve(packet);
					} else {
						handler.reject(packet);
					}
				} else {
					console.error('Missing handler for Websocket message REID('+packet.REID+')');
				}
			} else { // Stream
				const stream_data = this.stream_handler[packet.REID];
				const {handlers} = stream_data;
				if (handlers) {
					for(let i = 0; i<handlers.length; ++i) {
						const h = handlers[i];
						if (h.active) {
							h.handler(packet, stream_data.first);
						}
					}
					stream_data.first = false;
					this.last_stream_data[packet.REID] = packet;
				} else {
					console.error('Missing handler for Websocket message REID('+packet.REID+')');
				}
			}
		} catch(err) {
			console.error(err);
		}
		
	};
	
	ws.onerror = (error) => {
		console.error('websocket error', error);
	};
	ws.onclose = (event) => {
		this.closed = true;
		if (typeof this.onclose == 'function') {
			this.onclose();
		} else if (page_is_open) { alert('Connection closed. Reload to reconnect!'); }
		console.log('websocket close', event);
	};
	
};

LHWebSocket.prototype.wrap_payl_handler = function(f) {
	if (!f) return (() => null);
	return (packet => packet.RNUM === 200 ? f(packet.PAYL) : null);
}

LHWebSocket.prototype.decodeTarget = function(target) {
	const splitIndex = target.indexOf(':');
	if (splitIndex === -1) {
		console.error('Invalid LHWebsSocket Target(missing \':\'): '+target);
	}
	let method = target.slice(0, splitIndex);
	let path   = target.slice(splitIndex+1).split('/');
	return {
		method: method,
		path:   path
	};
}

LHWebSocket.prototype.sendObj = function(obj) {
	this.ws.send(msgpack.encode(obj, this.msgpack_options));
}

LHWebSocket.prototype.request = function(target, data) {
	const reid = this.free_RequestIDs.pop() || this.next_RequestID--;
	if (typeof target === 'string') {
		target = this.decodeTarget(target);
	}
	if (target.method === 'STREAM') {
		throw new Error('LHWebSocket Error: method STREAM not allowed in request');
	}
	return new Promise((resolve, reject) => {
		let packet = {
			VERB: target.method,
			PATH: target.path,
			AUTH: this.auth,
			META: {},
			PAYL: data,
			REID: reid
			};
		
		this.request_handler[-reid] = {resolve, reject, packet};
		
		this.sendObj(packet);
	});
}


LHWebSocket.prototype.stream = function(target, payl, handler) {
	if (typeof target === 'string') {
		if (target.indexOf(':') === -1) target = 'STREAM:'+target;
		target = this.decodeTarget(target);
	}
	if (target.method !== 'STREAM') {
		throw new Error('LHWebSocket Error: method is not STREAM in stream');
	}
	if (handler === undefined && typeof payl === 'function') {
		handler = payl;
		payl = null;
	}
	if (payl != null) {
		console.warn('WARNING: STREAM Payl != null', payl);
	}
	const ref = this.streammap[target.path.join('/')];
	if (ref == null || !ref.active) {
		const reid = ref == null ? this.next_StreamID++ : ref.reid;
		this.streammap[target.path.join('/')] = {reid, active: true};
		if (this.stream_handler[reid] != null) {
			const stream_data = this.stream_handler[reid];
			stream_data.first = true;
			if (handler != null) {
				const handler_obj = {handler, active: true};
				const {handlers} = stream_data;
				handlers[handlers.length] = handler_obj;
			}
		} else {
			this.stream_handler[reid] = {handlers: [{handler, active: (handler != null)}], first: true};
		}
		const data = {
			VERB: target.method,
			PATH: target.path,
			AUTH: this.auth,
			META: {},
			PAYL: payl,
			REID: reid
		};
		this.sendObj(data);
	} else { // stream exists
		const reid = ref.reid;
		const last_data = this.last_stream_data[reid];
		const handler_obj = {handler, active: true};
		const {handlers} = this.stream_handler[reid];
		handlers[handlers.length] = handler_obj;
		if (last_data !== undefined) {
			handler_obj.handler(last_data, true);
		}
	}
}




/*
function AnimManager(name, fps) {
	this.fps       = fps || 30;
	this.buffer    = new ArrayBuffer(3*392);
	this.img       = new Uint8Array(this.buffer);
	this.execute   = null;
	this.active    = true;
	this.uid       = Date.now() + '';
	this.stoppedBy = null;
	this.name      = name || 'DefaultName';
	this.otherManager = {};
	this.next = () => {
		if (this.active) {
			if (this.execute) this.execute(Date.now()/1000);
			lhws.request("PUT:user/"+auth.USER+"/model", this.buffer);
		}
		setTimeout(this.next, 1000/this.fps);
	};
	this.stop = () => {
		this.stoppedBy = null;
		if (this.active) {
			lhws.request("PUT:user/"+auth.USER+"/model", {type: 'animhandler', action: 'stopping', src: this.uid});
			this.active = false;
		}
	};
	this.activate = () => {
		if (!this.active) {
			lhws.request("PUT:user/"+auth.USER+"/model", {type: 'animhandler', action: 'start', src: this.uidi, target: this.uid});
		}
	};
	this.reinfo = () => {
		this.otherManager = {};
		lhws.request("PUT:user/"+auth.USER+"/model", {type: 'animhandler', action: 'info', src: this.uid});
	};
	
	lhws.stream("STREAM:user/"+auth.USER+"/model", null, (packet) => {
		const data = packet.PAYL;
		if (packet.RNUM === 200 && data instanceof Object && data.type === 'animhandler') {
			const fromMe = data.src === this.uid;
			if (data.action === 'stop' && !fromMe) {
				this.active = false;
			}
			if (data.action === 'start') {
				const targetMe = data.target === this.uid;
				if (this.active && !targetMe) {
					lhws.request("PUT:user/"+auth.USER+"/model", {type: 'animhandler', action: 'stopped', src: this.uid, by: data.src});
					this.stoppedBy = data.src;
					console.log('AnimManager stopped by other AnimManager');
				}
				this.active = targetMe;
				if (targetMe) {
					this.stoppedBy = null;
				}
			}
			if (data.action === 'stopped' && !fromMe && this.hasstopped == null && this.active) {
				this.hasstopped = data.src;
			}
			if (data.action === 'stopping' && data.src === this.stoppedBy && !this.active) {
				this.stoppedBy = null;
				this.active = true;
				console.log('AnimManager reactivated by termination of remote AnimManager');
				lhws.request("PUT:user/"+auth.USER+"/model", {type: 'animhandler', action: 'start', target: this.uid, src: this.uid, name: this.name});
			}
			if (data.action === 'info' && !fromMe) {
				lhws.request("PUT:user/"+auth.USER+"/model", {type: 'animhandler', action: 'list', src: this.uid, target: data.src, name: this.name});
				this.otherManager[data.src] = {id: data.src, name: data.name};
			}
			if (data.action === 'list') {
				this.otherManager[data.src] = {id: data.src, name: data.name};
			}
		}
	});
	lhws.request("PUT:user/"+auth.USER+"/model", {type: 'animhandler', action: 'start', target: this.uid, src: this.uid, name: this.name});
	lhws.request("PUT:user/"+auth.USER+"/model", {type: 'animhandler', action: 'info', src: this.uid, name: this.name});
	
	this.next();
}
*/

