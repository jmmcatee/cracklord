cracklord.directive("confirmPopover", function() {
    return {
        restrict: 'A',
        link: function (scope, el, attrs) {
            var id = scope.job.jobid;
            var html = '<div id="confirm_'+id+'" class="btn-group"> <button type="button" class="btn btn-small btn-success"> <i class="fa fa-check-circle"></i> Yes</button><button type="button" class="btn btn-small btn-danger"><i class="fa fa-times-circle"></i> No</button></div>';

            $(el).popover({
                trigger: 'manual',
                html: true,
                title: attrs.popoverText || "Are you sure?",
                content: html,
                placement: attrs.popoverPlacement || "top",
                container: 'body'
            });

            el.bind('click', function(e) {
                e.stopPropagation();
                el.popover('show');
                var popoverDiv = $("#confirm_"+id)
                popoverDiv.find(".btn-success").click(function(e) {
                    el.popover('hide');
                    scope.$apply(attrs.confirmPopover);
                });
                popoverDiv.find(".btn-danger").click(function(e) {
                    el.popover('hide');
                });
            });

        }
    };
});

cracklord.directive('rcSubmit', ['$parse', function ($parse) {
    return {
        restrict: 'A',
        require: ['rcSubmit', '?form'],
        controller: ['$scope', function ($scope) {
            this.attempted = false;
            
            var formController = null;
            
            this.setAttempted = function() {
                this.attempted = true;
            };
            
            this.setFormController = function(controller) {
              formController = controller;
            };
            
            this.needsAttention = function (fieldModelController) {
                if (!formController) return false;
                
                if (fieldModelController) {
                    return fieldModelController.$invalid && (fieldModelController.$dirty || this.attempted);
                } else {
                    return formController && formController.$invalid && (formController.$dirty || this.attempted);
                }
            };
        }],
        compile: function(cElement, cAttributes, transclude) {
            return {
                pre: function(scope, formElement, attributes, controllers) {
                  
                    var submitController = controllers[0];
                    var formController = (controllers.length > 1) ? controllers[1] : null;
                    
                    submitController.setFormController(formController);
                    
                    scope.rc = scope.rc || {};
                    scope.rc[attributes.name] = submitController;
                },
                post: function(scope, formElement, attributes, controllers) {
                  
                    var submitController = controllers[0];
                    var formController = (controllers.length > 1) ? controllers[1] : null;
                    var fn = $parse(attributes.rcSubmit);
                    
                    formElement.bind('submit', function (event) {
                        submitController.setAttempted();
                        if (!scope.$$phase) scope.$apply();
                        
                        if (!formController.$valid) return false;
                
                        scope.$apply(function() {
                            fn(scope, {$event:event});
                        });
                    });
                }
          };
        }
    };
}]);