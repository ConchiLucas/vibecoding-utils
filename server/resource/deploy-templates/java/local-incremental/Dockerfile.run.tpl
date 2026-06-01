FROM eclipse-temurin:{{ .JavaVersion }}-jre-alpine
WORKDIR /app
COPY target/*.jar app.jar
EXPOSE {{ .AppPort }}
ENTRYPOINT ["java", "-jar", "app.jar"]
