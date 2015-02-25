$(function () {
  $('[data-toggle="tooltip"]').tooltip()
})

var cracklord = angular.module('cracklord', [
	'ui.router',
    'ui.sortable',
    'readableTime'
]);

cracklord.directive('tooltip', function(){
    return {
        restrict: 'A',
        link: function(scope, element, attrs){
            $(element).hover(function(){
                // on mouseenter
                $(element).tooltip('show');
            }, function(){
                // on mouseleave
                $(element).tooltip('hide');
            });
        }
    };
});








