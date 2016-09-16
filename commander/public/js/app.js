


      function bytesToSize(bytes) {
          var sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
          if (bytes == 0) return 'n/a';
          var i = parseInt(Math.floor(Math.log(bytes) / Math.log(1024)));
          if (i == 0) return bytes + ' ' + sizes[i]; 
          return (bytes / Math.pow(1024, i)).toFixed(1) + ' ' + sizes[i];
      }


      function uploadFiles(event)
      {
          event.stopPropagation(); // Stop stuff happening
          event.preventDefault(); // Totally stop stuff happening

          // START A LOADING SPINNER HERE

          // Create a formdata object and add the files
          var data = new FormData();
          console.log(document.getElementById('file').files);
          $.each(document.getElementById('file').files, function(key, value)
          {
              data.append(key, value);
          });

          $.ajax({
              url: '/api/update_directives',
              type: 'POST',
              data: data,
              cache: false,
              dataType: 'json',
              processData: false, // Don't process the files
              contentType: false, // Set content type to false as jQuery will tell the server its a query string request
              beforeSend: function(evt) {
                console.log(evt);
              },
              success: function(data, textStatus, jqXHR)
              {
                  if(data.status == 'ok')
                  {
                      // Success so call function to process the form
                      //submitForm(event, data);
                      console.log('Successfully updated.');
                      //$('#directives_content').val(data);
                      $('#config_directives').html('<pre>'+JSON.stringify(data, null, "\t")+'</pre>');
                      alert("Notice: Directives successfully recieved.  Starting tests.");
                  }
                  else
                  {
                      // Handle errors here
                      console.log('ERRORS: ' + data.error);
                      alert("Error: "+data.message);
                  }
              },
              error: function(jqXHR, textStatus, errorThrown)
              {
                  // Handle errors here
                  console.log('ERRORS: ' + errorThrown);
                  console.log('ERRORS: ' + textStatus);
                  // STOP LOADING SPINNER
              }
          });
      }


      var refreshInterval = 2000;

      /********** BEGING OF document ready *********/
      $(document).ready(function(e) {
  
 
        var requests = [];
        var active_nodes = [];
        var tbl = $('#url_breakdown');
        var tbl_nodes = $('#nodes');


        $('#btn_halt').on('click', function(e){

          e.preventDefault();
          $.ajax({ 
            url: "http://localhost:8082/api/halt", 
            success: function(data){
              alert(data['message']);
            }, 
            dataType: "json"
          });

        });

        $('#btn_shutdown').on('click', function(e){

          e.preventDefault();
          $.ajax({ 
            url: "http://localhost:8082/api/shutdown", 
            success: function(data){
              alert(data['message']);
            }, 
            dataType: "json"
          });

        });
        /* ********************************** */


        $('#upload_directives').on('submit', uploadFiles);

        setInterval(function(){
            $.ajax({ 
                url: "http://localhost:8082/api/stats", 
                success: function(data){
                  
                  $('#requests_completed').html(data.total_requests_completed);
                  $('#data_transfered').html(bytesToSize(data.total_bytes_downloaded));
                  $('#rps').html(data.global_rps);
                  $('#rps_failed').html(data.global_rps_failed);
                  $('#requests_failed').html(data.total_requests_failed);

                  requests = [];
                  for(var key in data.request_breakdown) {
                    parts = key.split("~",2);
                    var elem = data.request_breakdown[key];
                    requests.push({
                      "hits": elem.total_hits,
                      "url": parts[1],
                      "method": parts[0],
                      "last_response_code": elem.last_response_code,
                      "last_response_size": elem.last_response_size
                    });
                  }

                  tbl.dynatable({
                    dataset: {
                      records: requests
                    }
                  });

                  /*
                  dynatable.settings.dataset.originalRecords = myRecords;
                  dynatable.process();
                  */
                  tbl.data('dynatable').settings.dataset.records = requests;
                  tbl.data('dynatable').dom.update();
                  /*
                  tbl.data('dynatable').sorts.clear();
                  tbl.data('dynatable').sorts.add('hits', 1); // 1=ASCENDING, -1=DESCENDING
                  tbl.data('dynatable').process();
                  */

                }, 
                dataType: "json"});
        }, refreshInterval);

        setInterval(function(){
            $.ajax({ 
                url: "http://localhost:8082/api/current_nodes", 
                success: function(data){
                  
                  var total_nodes = 0;
                  active_nodes = [];

                  var dt = new Date();
                  var now;

                  for(var id in data) {
                    
                    now = dt.getTime()
                    total_nodes+=1;
                    var d = new Date(data[id]['last_checkin']);
                    /*
                    Right now the last_checkin threshold is set statically in this code
                    although it should be moved to the commander INI file, and then provided to
                    the front-end via the "GET /configuration" API call.
                    */
                    active_nodes.push({
                      "hostname": data[id]['hostname'],
                      "id": id,
                      "status": (now-data[id]['last_checkin'] <= (300*1000) ? 'OK' : 'GONE'),
                      "last_checkin": d.getFullYear()+"-"+d.getMonth()+"-"+d.getDate()+" "+d.getHours()+":"+d.getMinutes()+":"+d.getSeconds()
                    })

                  }

                  $('#total_nodes').html(total_nodes);
                  tbl_nodes.dynatable({
                    dataset: {
                      records: active_nodes
                    }
                  });
                  tbl_nodes.data('dynatable').settings.dataset.records = active_nodes;
                  tbl_nodes.data('dynatable').dom.update();


                }, 
                dataType: "json"});
        }, refreshInterval);
  
        /**********************************/
        $.ajax({ 
          url: "http://localhost:8082/api/configs/commander", 
          success: function(data){
            $('#config_commander').html('<pre>'+JSON.stringify(data, null, "\t")+'</pre>');
          }, 
          dataType: "json"
        });

        $.ajax({ 
          url: "http://localhost:8082/api/configs/node", 
          success: function(data){
            $('#config_node').html('<pre>'+JSON.stringify(data, null, "\t")+'</pre>');
          }, 
          dataType: "json"
        });
      /********** END OF document ready *********/
      });
