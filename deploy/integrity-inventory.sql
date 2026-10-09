-- Read-only pre-deployment inventory. Run with psql -v ON_ERROR_STOP=1.
SELECT conrelid::regclass AS relation, conname, convalidated, pg_get_constraintdef(oid)
FROM pg_constraint WHERE connamespace='public'::regnamespace AND contype IN ('f','c') ORDER BY 1,2;
SELECT 'rating_score' AS issue,count(*) FROM chofer_calificaciones WHERE puntaje NOT BETWEEN 1 AND 5
UNION ALL SELECT 'rating_aggregate',count(*) FROM chofers WHERE rating_count < 0 OR rating_average NOT BETWEEN 0 AND 5
UNION ALL SELECT 'network_company',count(*) FROM red_choferes r LEFT JOIN empresas e ON e.id=r.empresa_id WHERE e.id IS NULL
UNION ALL SELECT 'network_driver',count(*) FROM red_choferes r LEFT JOIN chofers c ON c.id=r.chofer_id WHERE c.id IS NULL
UNION ALL SELECT 'interest_load',count(*) FROM load_interests r LEFT JOIN cargas c ON c.id=r.carga_id WHERE c.id IS NULL
UNION ALL SELECT 'interest_driver',count(*) FROM load_interests r LEFT JOIN chofers c ON c.id=r.chofer_id WHERE c.id IS NULL
UNION ALL SELECT 'rating_load',count(*) FROM chofer_calificaciones r LEFT JOIN cargas c ON c.id=r.carga_id WHERE c.id IS NULL
UNION ALL SELECT 'rating_driver',count(*) FROM chofer_calificaciones r LEFT JOIN chofers c ON c.id=r.chofer_id WHERE c.id IS NULL
UNION ALL SELECT 'rating_publisher',count(*) FROM chofer_calificaciones r LEFT JOIN publicadors p ON p.id=r.publicador_id WHERE p.id IS NULL;
