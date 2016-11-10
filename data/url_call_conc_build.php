<?php
/**
 * for url_call_conc
 */
for($i=0;$i<200;$i++){
    $hd=array(
        "url"=>"http://127.0.0.1:8000/test/a.php?i=".$i,
        "method"=>"post",
        "header"=>array(
            "Content-Type"=>"application/x-www-form-urlencoded",
         ),
    );
    $hd=json_encode($hd);
    $post=array("post_id"=>$i);
    $bd=http_build_query($post);
    echo sprintf("%d|%s%d|%s",strlen($hd),$hd,strlen($bd),$bd);
}