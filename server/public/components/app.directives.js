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