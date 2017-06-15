// Copyright 2017 The tcp-srv Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

const indexTmpl = `
<html>
	<head>
		<title>SoLiD sensors monitoring</title>
		<script type="text/javascript">
		var sock = null;

		function update(data) {
			var p = null;
			
			p = document.getElementById("sensor-plot");
			p.innerHTML = data.plot;

			p = document.getElementById("update-message");
			p.innerHTML = "Last Update: <code>"+data.update+"</code>";
		};

		window.onload = function() {
			sock = new WebSocket("ws://"+location.host+"/data");
			sock.onmessage = function(event) {
				var data = JSON.parse(event.data);
				update(data);
			};
		};
		</script>

		<style>
		.solid-plot-style {
			font-size: 14px;
			line-height: 1.2em;
		}
		</style>
	</head>

	<body>
		<div id="header">
			<h2>SoLiD sensors monitoring plots</h2>
		</div>

		<div id="plots">
			<div id="sensor-plot" class="solid-plot-style"></div>
		</div>
		<br>
		<div id="update-message">Last Update: N/A
		</div>
	</body>
</html>
`
