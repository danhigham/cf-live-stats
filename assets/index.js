(function() {

  var parseTime = d3.time.format("%Y-%m-%d %H:%M:%S +0000 +0000").parse;
  var color = d3.scale.category10();

  function Chart (target, metricFilter, yAxisLabel, yMetricMultiplier) {

    var m = yMetricMultiplier || 1;
    var _self = this;
    var data = null;

    target = d3.select(target);
    var nodeWidth = target.node().clientWidth;

    var margin = {top: 20, right: 80, bottom: 30, left: 50},
    width = nodeWidth - margin.left - margin.right,
    height = 500 - margin.top - margin.bottom;

    var x = d3.time.scale()
    .range([0, width]);

    var y = d3.scale.linear()
    .range([height, 0]);

    var xAxis = d3.svg.axis()
    .scale(x)
    .orient("bottom");

    var yAxis = d3.svg.axis()
    .scale(y)
    .orient("left");

    var line = d3.svg.line()
    .interpolate("basis")
    .x(function(d) { return x(d.time); })
    .y(function(d) { return y(d.metricValue * m); });

    var svg = target.append("svg")
    .attr("width", width + margin.left + margin.right)
    .attr("height", height + margin.top + margin.bottom)
    .append("g")
    .attr("transform", "translate(" + margin.left + "," + margin.top + ")");

    this.setSVGDimensions = function () {

      nodeWidth = target.node().clientWidth;

      margin = {top: 20, right: 80, bottom: 30, left: 50};
      width = nodeWidth - margin.left - margin.right;
      height = 500 - margin.top - margin.bottom;

      target.select("svg")
      .attr("width", width + margin.left + margin.right)
      .attr("height", height + margin.top + margin.bottom);

      x = d3.time.scale()
      .range([0, width]);

      xAxis = d3.svg.axis()
      .scale(x)
      .orient("bottom");

    };

    this.getData = function () {
      return this.data;
    };

    this.svg = function () {
      return this.svg;
    };

    this.update = function(data) {

      this.data = data;

      svg.selectAll("*").remove();

      var metrics = color.domain().filter(function(key) { return key.substr(0, metricFilter.length) == metricFilter; }).map(function(name) {
        return {
          name: name,
          values: data.map(function(d) {
            return {time: d.time, metricValue: +d[name]};
          })
        };
      });

      x.domain(d3.extent(data, function(d) { return d.time; }));

      y.domain([0, d3.max(metrics, function(c) { return d3.max(c.values, function(v) { return v.metricValue * m; }); }) ]);

      svg.append("g")
      .attr("class", "x axis")
      .attr("transform", "translate(0," + height + ")")
      .call(xAxis);

      svg.selectAll("line.horizontalGrid").data(y.ticks(8)).enter()
      .append("line")
      .attr({
        "class":"horizontalGrid",
        "x1" : 0,
        "x2" : width,
        "y1" : function(d){ return y(d);},
        "y2" : function(d){ return y(d);},
        "fill" : "none",
        "shape-rendering" : "crispEdges",
        "stroke-width" : "1px"
      });

      svg.selectAll("line.verticalGrid").data(x.ticks(8)).enter()
      .append("line")
      .attr({
        "class":"verticalGrid",
        "x1" : function(d){ return x(d);},
        "x2" : function(d){ return x(d);},
        "y1" : 0,
        "y2" : height,
        "fill" : "none",
        "shape-rendering" : "crispEdges",
        "stroke-width" : "1px"
      });

      svg.append("g")
      .attr("class", "y axis")
      .call(yAxis)
      .append("text")
      .attr("transform", "rotate(-90)")
      .attr("y", 6)
      .attr("dy", ".71em")
      .style("text-anchor", "end")
      .text(yAxisLabel);

      var metric = svg.selectAll(".metric")
      .data(metrics)
      .enter().append("g")
      .attr("class", "metric");

      metric.append("path")
      .attr("class", "line")
      .attr("d", function(d) { return line(d.values); })
      .style("stroke", function(d) { return color(d.name); });

      metric.append("text")
      .datum(function(d) { return {name: d.name, value: d.values[d.values.length - 1]}; })
      .attr("transform", function(d) { return "translate(" + x(d.value.time) + "," + y(d.value.metricValue * m) + ")"; })
      .attr("x", 3)
      .attr("dy", ".35em")
      .text(function(d) { return d.name; });
    };

    return this;
  }

  function pullData() {

    d3.json("data.json", function(error, data) {

      color.domain(d3.keys(data[0]).filter(function(key) { return key !== "time"; }));

      data.sort(function(a, b){ return d3.ascending(a.time, b.time); });

      data.forEach(function(d) {
        d.time = parseTime(d.time);
        // TODO : Convert to local time
      });

      cpuChart.update(data);
      memChart.update(data);

    });

  }

  var cpuChart = new Chart("#cpu-chart", "cpu", "CPU %", 100);
  var memChart = new Chart("#mem-chart", "mem", "Memory (MB)", 1 / 1048576);

  d3.timer(function() { pullData(); }, 3000);

  pullData();

  d3.select(window).on('resize', function () {

    cpuChart.setSVGDimensions();
    cpuChart.update(cpuChart.getData());

    memChart.setSVGDimensions();
    memChart.update(cpuChart.getData());

  });


})();
