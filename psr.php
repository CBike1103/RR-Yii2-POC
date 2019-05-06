<?php
ini_set('display_errors', 'stderr');
include "vendor/autoload.php";

$relay = new Spiral\Goridge\StreamRelay(STDIN, STDOUT);
$psr7 = new Spiral\RoadRunner\PSR7Client(new Spiral\RoadRunner\Worker($relay));

while ($req = $psr7->acceptRequest()) {
    try {
        $resp = new \Zend\Diactoros\Response();
        ob_start();
        phpinfo();
        $variable = ob_get_clean();
        $resp->getBody()->write($variable);

        $psr7->respond($resp);
    } catch (\Throwable $e) {
        $psr7->getWorker()->error((string) $e);
    }
}
