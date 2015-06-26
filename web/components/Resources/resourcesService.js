cracklord.factory('ResourceService', ['$resource', function ($resource) {
   return $resource('/api/resources/:id', {id: '@id'}, {
   	query: {
   		isArray: true,
   		method: 'GET',
   		params: {},
   		transformResponse: function(data) {
   			var results = angular.fromJson(data);
   			return results.resources;
   		}	
   	},
      update: {
         method: 'PUT', 
         transformResponse: function(data) {
            var results = angular.fromJson(data);
            return results.resource;
         }
      }
   });
}]);

cracklord.service('ResourceList', ['ResourceService', 'ResourceColorizer', '$filter', function(ResourceService, ResourceColorizer, $filter) {
   var colorizer = ResourceColorizer;
   var resources = {};
   resources.list = [];

   resources.update = function() {
      return ResourceService.query(
         function(data) {
            for(var i = 0; i < resources.list.length; i++) {
               for(var j = 0; j < data.length; j++) {
                  if(data[j].id === resources.list[i].id) {
                     for(var prop in data[j]) {
                        if(data[j].hasOwnProperty(prop)) {
                           resources.list[i][prop] = data[j][prop]
                        }  
                     }
                  }
               } 
            }
         }
      );
   }

   resources.load = function() {
      return ResourceService.query(
         function(data) {
            colorizer.resetSeed()
            for(var i = 0; i < data.length; i++) {
               data[i].color = colorizer.getColor();
            }
            angular.copy(data, resources.list)
         }
      );
   }

   resources.get = function(id) {
      var found = $filter('filter')(resources.list, {id: id}, true);
      if(found) {
         return found[0];
      } else {
         return false;
      }
   }

   return resources;
}]);

cracklord.service('ResourceColorizer', function() {
   var seed = 0.54;

   var HUEtoRGB = function (p, q, t){
      if(t < 0) t += 1;
      if(t > 1) t -= 1;
      if(t < 1/6) return p + (q - p) * 6 * t;
      if(t < 1/2) return q;
      if(t < 2/3) return p + (q - p) * (2/3 - t) * 6;
      return p;
   }

   var HSVtoRGB = function (h, s, v) {
      var r, g, b;

      if(s == 0) {
         r = g = b = v;
      } else {
         var q = v < 0.5 ? v * (1 + s) : v + s - v * s;
         var p = 2 * v - q;
         r = HUEtoRGB(p, q, h + 1/3);
         g = HUEtoRGB(p, q, h);
         b = HUEtoRGB(p, q, h - 1/3);
      }

      return {
         'r': Math.round(r * 255), 
         'g': Math.round(g * 255), 
         'b': Math.round(b * 255)
      };
   }

   this.resetSeed = function() {
      seed = 0.54;
   }

   this.getColor = function() {
      seed += 0.618033988749895;
      seed %= 1;
      return HSVtoRGB(seed, 0.5, 0.95);
   }

   return this;
});